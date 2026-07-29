package router

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/qpubio/qpub-server/internal/domain/messaging/backpressure"
	"github.com/qpubio/qpub-server/internal/domain/messaging/broker"
	"github.com/qpubio/qpub-server/internal/domain/messaging/channel"
	"github.com/qpubio/qpub-server/internal/domain/messaging/delivery"
	"github.com/qpubio/qpub-server/internal/domain/messaging/envelope"
	domainRouter "github.com/qpubio/qpub-server/internal/domain/messaging/router"
	"github.com/qpubio/qpub-server/internal/domain/messaging/protocol"
	"github.com/qpubio/qpub-server/internal/domain/messaging/publication"
	"github.com/qpubio/qpub-server/internal/domain/messaging/receipt"
	"github.com/qpubio/qpub-server/internal/domain/messaging/subscription"
	domainTelemetry "github.com/qpubio/qpub-server/internal/domain/messaging/telemetry"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

const subscriptionBufferCapacity = 256

type subscriptionBufferState struct {
	queue  *backpressure.BoundedQueue
	policy backpressure.Policy
}

// Service routes inbound publishes and broker deliveries through the runtime.
type Service struct {
	logger           logger.Service
	instanceID       id.ULID
	brokerService    broker.Service
	channelService   channel.Service
	subscriptionRepo subscription.Repository
	deliverer        delivery.Deliverer
	telemetryService domainTelemetry.Service
	gatekeeper       backpressure.Gatekeeper

	subBuffersMu sync.Mutex
	subBuffers   map[id.ULID]*subscriptionBufferState
	subDropPolicy backpressure.Policy
}

// NewService creates a message router.
func NewService(
	logger logger.Service,
	instanceID id.ULID,
	brokerService broker.Service,
	channelService channel.Service,
	subscriptionRepo subscription.Repository,
	deliverer delivery.Deliverer,
	telemetryService domainTelemetry.Service,
	gatekeeper backpressure.Gatekeeper,
) domainRouter.Service {
	return &Service{
		logger:           logger,
		instanceID:       instanceID,
		brokerService:    brokerService,
		channelService:   channelService,
		subscriptionRepo: subscriptionRepo,
		deliverer:        deliverer,
		telemetryService: telemetryService,
		gatekeeper:       gatekeeper,
		subBuffers:       make(map[id.ULID]*subscriptionBufferState),
		subDropPolicy:    backpressure.NewDropNewestPolicy(backpressure.PointSubscriptionBuffer),
	}
}

// PublishInbound accepts, validates, and routes an inbound publish.
func (s *Service) PublishInbound(ctx context.Context, req domainRouter.PublishRequest) (*receipt.Receipt, *publication.PublishResult, error) {
	if req.Message == nil {
		return receipt.IngressNack(id.ULID(""), "message is required", protocol.ErrInvalidMessage),
			nil, fmt.Errorf("message is required")
	}

	if s.gatekeeper != nil && !req.SkipTelemetry && !s.gatekeeper.AllowInbound(req.ProjectID) {
		s.recordInboundRejected(req.ProjectID, req.ConnectionID, req.Channel)
		return receipt.IngressNack(id.ULID(""), "inbound rate limit exceeded", protocol.ErrRateLimited),
			nil, publication.ErrRateLimited
	}

	ch, err := s.channelService.GetOrCreate(req.Channel, req.ProjectID)
	if err != nil {
		s.logger.Error(log.MessagingPublication, "Failed to get or create channel channel=%s projectID=%d error=%v",
			req.Channel, req.ProjectID, err)
		return receipt.IngressNack(id.ULID(""), "failed to get or create channel", protocol.ErrInternal),
			nil, fmt.Errorf("failed to get or create channel: %w", err)
	}

	if err := s.EnsureChannelListening(ch.FullName()); err != nil {
		s.logger.Error(log.MessagingPublication, "Failed to ensure broker listener channel=%s error=%v",
			ch.FullName(), err)
		return receipt.IngressNack(id.ULID(""), "failed to set up channel listener", protocol.ErrInternal),
			nil, fmt.Errorf("failed to ensure channel listening: %w", err)
	}

	payloads := make([]protocol.DataMessagePayload, len(req.Message.Messages))
	for i, msg := range req.Message.Messages {
		payloads[i] = *protocol.NewDataMessagePayload(msg.Alias, msg.Event, msg.Data)
	}

	dataMsg := protocol.NewDataMessage(
		protocol.ActionMessage,
		req.Channel,
		payloads,
		nil,
	)

	messageBytes, err := json.Marshal(dataMsg)
	if err != nil {
		return receipt.IngressNack(dataMsg.ID, "invalid message format", protocol.ErrInvalidMessage),
			nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	env := envelope.New(envelope.CreateParams{
		Direction:     envelope.DirectionInbound,
		ProjectID:     req.ProjectID,
		Channel:       ch.FullName(),
		Payload:       messageBytes,
		Source:        req.Source,
		ConnectionID:  req.ConnectionID,
		SkipTelemetry: req.SkipTelemetry,
	})
	env = env.WithID(dataMsg.ID)

	if err := s.brokerService.PublishToChannel(ch.FullName(), messageBytes); err != nil {
		s.logger.Error(log.MessagingPublication, "Failed to publish to broker channel=%s fullName=%s error=%v",
			req.Channel, ch.FullName(), err)
		return receipt.IngressNack(env.ID(), "failed to publish message", protocol.ErrPublishFailed),
			nil, fmt.Errorf("failed to publish message: %w", err)
	}

	result := &publication.PublishResult{
		MessageID:    dataMsg.ID,
		PublishedAt:  dataMsg.Timestamp,
		Channel:      req.Channel,
		PayloadCount: len(payloads),
	}

	if !req.SkipTelemetry {
		evt := domainTelemetry.NewEvent(
			domainTelemetry.TypeInboundAccepted,
			req.ProjectID,
			s.instanceID,
			domainTelemetry.InboundAcceptedData{
				EnvelopeID:   env.ID(),
				ConnectionID: req.ConnectionID,
				Channel:      req.Channel,
				ByteSize:     int64(len(messageBytes)),
				Source:       string(req.Source),
			},
		)
		if err := s.telemetryService.Record(evt); err != nil {
			s.logger.Warn(log.MessagingPublication, "Failed to record inbound telemetry connectionID=%s channel=%s error=%v",
				req.ConnectionID, req.Channel, err)
		}
	}

	s.logger.Debug(log.MessagingPublication, "Published message channel=%s fullName=%s messageCount=%d",
		req.Channel, ch.FullName(), len(req.Message.Messages))

	return receipt.IngressAck(env.ID()), result, nil
}

// OnBrokerMessage fans out a broker delivery to local subscribers.
func (s *Service) OnBrokerMessage(ctx context.Context, env *envelope.Envelope) error {
	if env == nil || len(env.Payload()) == 0 {
		return nil
	}

	fullChannelName := env.Channel()
	if fullChannelName == "" {
		return nil
	}

	s.touchChannelActivity(fullChannelName)

	subs, err := s.subscriptionRepo.GetAllLocalForChannel(fullChannelName)
	if err != nil {
		s.logger.Error(log.MessagingPublication, "Failed to get subscribers channel=%s error=%v",
			fullChannelName, err)
		return err
	}

	delivered := 0
	for _, sub := range subs {
		if sub == nil || sub.IsClosed() {
			continue
		}
		if s.deliverToSubscriber(sub, env.Payload(), env.ProjectID()) {
			delivered++
		}
	}

	s.logger.Info(log.MessagingPublication, "Broker message fanout complete channel=%s subscriberCount=%d delivered=%d",
		fullChannelName, len(subs), delivered)

	return nil
}

func (s *Service) deliverToSubscriber(sub *subscription.Subscription, payload []byte, projectID id.Int) bool {
	if s.gatekeeper != nil && !s.gatekeeper.AllowOutbound(projectID) {
		s.recordOutboundDrop(projectID, sub, backpressure.DropReasonOutboundRateLimit, 0)
		return false
	}

	buf := s.subscriptionBuffer(sub.ID())
	env := envelope.New(envelope.CreateParams{
		Direction: envelope.DirectionOutbound,
		ProjectID: projectID,
		Payload:   payload,
		Source:    envelope.SourceNATS,
	})

	result := s.subDropPolicy.TryEnqueue(buf.queue, env)
	if result.Dropped {
		s.recordOutboundDrop(projectID, sub, result.Reason, result.QueueDepth)
		return false
	}

	s.flushSubscriptionBuffer(sub)
	return true
}

func (s *Service) flushSubscriptionBuffer(sub *subscription.Subscription) {
	buf := s.subscriptionBuffer(sub.ID())
	for {
		item, ok := buf.queue.Pop()
		if !ok {
			return
		}
		if err := s.deliverer.Deliver(sub.ClientID(), item); err != nil {
			s.logger.Warn(log.MessagingPublication, "Failed to deliver buffered message subscriptionID=%s clientID=%s error=%v",
				sub.ID(), sub.ClientID(), err)
			return
		}
	}
}

func (s *Service) subscriptionBuffer(subID id.ULID) *subscriptionBufferState {
	s.subBuffersMu.Lock()
	defer s.subBuffersMu.Unlock()

	state, ok := s.subBuffers[subID]
	if ok {
		return state
	}

	state = &subscriptionBufferState{
		queue:  backpressure.NewBoundedQueue(subscriptionBufferCapacity),
		policy: s.subDropPolicy,
	}
	s.subBuffers[subID] = state
	return state
}

func (s *Service) recordInboundRejected(projectID id.Int, connectionID id.ULID, channel string) {
	s.logger.Warn(log.MessagingPublication, "Inbound publish rejected: rate limit projectID=%d connectionID=%s channel=%s",
		projectID, connectionID, channel)

	evt := domainTelemetry.NewEvent(
		domainTelemetry.TypeInboundRejected,
		projectID,
		s.instanceID,
		domainTelemetry.InboundRejectedData{
			Channel: channel,
			Reason:  string(backpressure.DropReasonInboundRateLimit),
		},
	)
	if err := s.telemetryService.Record(evt); err != nil {
		s.logger.Warn(log.MessagingPublication, "Failed to record inbound rejection telemetry projectID=%d error=%v",
			projectID, err)
	}
}

func (s *Service) recordOutboundDrop(projectID id.Int, sub *subscription.Subscription, reason backpressure.DropReason, queueDepth int) {
	s.logger.Warn(log.MessagingPublication, "Outbound message dropped subscriptionID=%s clientID=%s projectID=%d reason=%s queueDepth=%d",
		sub.ID(), sub.ClientID(), projectID, reason, queueDepth)

	evt := domainTelemetry.NewEvent(
		domainTelemetry.TypeOutboundDropped,
		projectID,
		s.instanceID,
		domainTelemetry.OutboundDroppedData{
			SubscriptionID: sub.ID(),
			Reason:         reason,
			QueueDepth:     queueDepth,
		},
	)
	if err := s.telemetryService.Record(evt); err != nil {
		s.logger.Warn(log.MessagingPublication, "Failed to record outbound drop telemetry subscriptionID=%s error=%v",
			sub.ID(), err)
	}
}

// DeliverOutbound enqueues an outbound payload for a subscription's client.
func (s *Service) DeliverOutbound(ctx context.Context, env *envelope.Envelope, subscriptionID id.ULID, connectionID id.ULID) (*receipt.Receipt, error) {
	if env == nil {
		return receipt.EgressDropped(id.ULID(""), "missing envelope"), fmt.Errorf("envelope is required")
	}

	sub, err := s.subscriptionRepo.FindByID(subscriptionID)
	if err != nil || sub == nil || sub.IsClosed() {
		return receipt.EgressDropped(env.ID(), "subscription not found"), fmt.Errorf("subscription not found")
	}

	if s.gatekeeper != nil && !s.gatekeeper.AllowOutbound(env.ProjectID()) {
		s.recordOutboundDrop(env.ProjectID(), sub, backpressure.DropReasonOutboundRateLimit, 0)
		return receipt.EgressDropped(env.ID(), string(backpressure.DropReasonOutboundRateLimit)), publication.ErrRateLimited
	}

	if err := s.deliverer.Deliver(sub.ClientID(), env.Payload()); err != nil {
		return receipt.EgressDropped(env.ID(), err.Error()), err
	}

	return receipt.EgressDelivered(env.ID()), nil
}

// EnsureChannelListening registers a broker listener for cross-instance fan-in.
func (s *Service) EnsureChannelListening(fullChannelName string) error {
	err := s.brokerService.ListenToChannel(fullChannelName, func(channelName string, message []byte) {
		parsed, parseErr := channel.FromFull(channelName)
		if parseErr != nil {
			s.logger.Error(log.MessagingBroker, "Failed to parse broker channel name channel=%s error=%v",
				channelName, parseErr)
			return
		}

		env := envelope.New(envelope.CreateParams{
			Direction: envelope.DirectionOutbound,
			ProjectID: parsed.ProjectID(),
			Channel:   channelName,
			Payload:   message,
			Source:    envelope.SourceNATS,
		})

		if err := s.OnBrokerMessage(context.Background(), env); err != nil {
			s.logger.Error(log.MessagingBroker, "Failed to fan out broker message channel=%s error=%v",
				channelName, err)
		}
	})
	if err != nil {
		return err
	}

	s.logger.Info(log.MessagingBroker, "Broker listener registered channel=%s instanceID=%s",
		fullChannelName, s.instanceID)

	return nil
}

func (s *Service) touchChannelActivity(fullChannelName string) {
	parsed, err := channel.FromFull(fullChannelName)
	if err != nil {
		return
	}

	ch, err := s.channelService.Get(parsed.Raw(), parsed.ProjectID())
	if err != nil {
		return
	}

	ch.UpdateActivity()
	_ = s.channelService.Update(ch)
}
