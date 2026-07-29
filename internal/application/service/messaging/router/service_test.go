package router_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	routerApp "github.com/qpubio/qpub-server/internal/application/service/messaging/router"
	"github.com/qpubio/qpub-server/internal/domain/messaging/broker"
	"github.com/qpubio/qpub-server/internal/domain/messaging/channel"
	"github.com/qpubio/qpub-server/internal/domain/messaging/envelope"
	"github.com/qpubio/qpub-server/internal/domain/messaging/protocol"
	domainRouter "github.com/qpubio/qpub-server/internal/domain/messaging/router"
	domainPublication "github.com/qpubio/qpub-server/internal/domain/messaging/publication"
	domainTelemetry "github.com/qpubio/qpub-server/internal/domain/messaging/telemetry"
	"github.com/qpubio/qpub-server/internal/domain/messaging/subscription"
	"github.com/qpubio/qpub-server/internal/domain/messaging/testsupport"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"github.com/stretchr/testify/require"
)

type nopLogger struct{}

func (nopLogger) Debug(log.Component, string, ...interface{}) {}
func (nopLogger) Info(log.Component, string, ...interface{})  {}
func (nopLogger) Warn(log.Component, string, ...interface{})  {}
func (nopLogger) Error(log.Component, string, ...interface{}) {}
func (nopLogger) Fatal(log.Component, string, ...interface{}) {}

type stubBroker struct {
	handlers map[string]broker.MessageHandler
}

func newStubBroker() *stubBroker {
	return &stubBroker{handlers: make(map[string]broker.MessageHandler)}
}

func (s *stubBroker) ListenToChannel(channelName string, handler broker.MessageHandler) error {
	s.handlers[channelName] = handler
	return nil
}
func (s *stubBroker) StopListeningToChannel(string) error { return nil }
func (s *stubBroker) PublishToChannel(subject string, message []byte) error {
	if handler, ok := s.handlers[subject]; ok {
		handler(subject, append([]byte(nil), message...))
	}
	return nil
}
func (s *stubBroker) Shutdown(context.Context) error { return nil }

type stubChannelSvc struct {
	instanceID id.ULID
}

func (s *stubChannelSvc) Create(string, id.Int) (*channel.Channel, error) { panic("unexpected") }
func (s *stubChannelSvc) Update(*channel.Channel) error                     { return nil }
func (s *stubChannelSvc) Delete(string, id.Int) error                      { panic("unexpected") }
func (s *stubChannelSvc) Get(rawName string, projID id.Int) (*channel.Channel, error) {
	return channel.New(channel.NewName(rawName, projID), s.instanceID), nil
}
func (s *stubChannelSvc) GetOrCreate(rawName string, projID id.Int) (*channel.Channel, error) {
	return channel.New(channel.NewName(rawName, projID), s.instanceID), nil
}

type memorySubscriptionRepo struct {
	mu       sync.Mutex
	channels map[string][]*subscription.Subscription
}

func newMemorySubscriptionRepo() *memorySubscriptionRepo {
	return &memorySubscriptionRepo{channels: make(map[string][]*subscription.Subscription)}
}

func (r *memorySubscriptionRepo) AddToChannel(channelName string, sub *subscription.Subscription) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.channels[channelName] = append(r.channels[channelName], sub)
	return nil
}
func (r *memorySubscriptionRepo) RemoveFromChannel(string, *subscription.Subscription) error {
	return nil
}
func (r *memorySubscriptionRepo) RemoveSubscriptionFromAll(*subscription.Subscription) error { return nil }
func (r *memorySubscriptionRepo) GetAllLocalForChannel(channelName string) ([]*subscription.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*subscription.Subscription(nil), r.channels[channelName]...), nil
}
func (r *memorySubscriptionRepo) GetAllChannelsForSubscription(*subscription.Subscription) ([]string, error) {
	return nil, nil
}
func (r *memorySubscriptionRepo) CleanClosedSubscriptions() int { return 0 }
func (r *memorySubscriptionRepo) FindByID(subID id.ULID) (*subscription.Subscription, error) {
	return nil, errors.New("not found")
}

type stubTelemetry struct{}

func (stubTelemetry) Record(*domainTelemetry.Event) error { return nil }
func (stubTelemetry) GetSnapshots(id.ULID) ([]*domainTelemetry.Snapshot, error) {
	return nil, nil
}
func (stubTelemetry) ResetForProject(id.Int) error { return nil }

type allowAllGatekeeper struct{}

func (allowAllGatekeeper) AllowInbound(id.Int) bool  { return true }
func (allowAllGatekeeper) AllowOutbound(id.Int) bool { return true }

type capturingDeliverer struct {
	mu       sync.Mutex
	messages map[id.ULID][][]byte
}

func newCapturingDeliverer() *capturingDeliverer {
	return &capturingDeliverer{messages: make(map[id.ULID][][]byte)}
}

func (d *capturingDeliverer) Deliver(clientID id.ULID, payload []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages[clientID] = append(d.messages[clientID], append([]byte(nil), payload...))
	return nil
}

func (d *capturingDeliverer) count(clientID id.ULID) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.messages[clientID])
}

func TestPublishInbound_FanoutToSubscribers(t *testing.T) {
	testsupport.InitRuntime()

	instanceID := id.NewULID()
	projectID := id.Int(7)
	rawChannel := "room-a"
	fullChannel := channel.NewName(rawChannel, projectID).Full()

	br := newStubBroker()
	repo := newMemorySubscriptionRepo()
	deliverer := newCapturingDeliverer()

	clients := []id.ULID{id.NewULID(), id.NewULID(), id.NewULID()}
	for _, clientID := range clients {
		sub := subscription.New(clientID, 16)
		require.NoError(t, repo.AddToChannel(fullChannel, sub))
	}

	router := routerApp.NewService(
		nopLogger{},
		instanceID,
		br,
		&stubChannelSvc{instanceID: instanceID},
		repo,
		deliverer,
		stubTelemetry{},
		allowAllGatekeeper{},
	)

	ev := "evt"
	msg, err := domainPublication.Create(domainPublication.CreateParams{
		ProjectID:   projectID,
		ChannelName: rawChannel,
		Messages: []domainPublication.Payload{{
			Event: &ev,
			Data:  map[string]string{"k": "v"},
		}},
	})
	require.NoError(t, err)

	rcpt, result, err := router.PublishInbound(context.Background(), domainRouter.PublishRequest{
		ProjectID: projectID,
		Channel:   rawChannel,
		Message:   msg,
		Source:    envelope.SourceREST,
	})
	require.NoError(t, err)
	require.True(t, rcpt.IsAck())
	require.NotNil(t, result)

	for _, clientID := range clients {
		require.Equal(t, 1, deliverer.count(clientID), "client %s should receive one message", clientID)
	}

	var wire protocol.DataMessage
	require.NoError(t, json.Unmarshal(deliverer.messages[clients[0]][0], &wire))
	require.Equal(t, protocol.ActionMessage, wire.Action)
	require.Equal(t, result.MessageID, wire.ID)
}
