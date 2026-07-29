package session

import (
	"sync"

	"github.com/qpubio/qpub-server/internal/application/service/messaging/egress"
	"github.com/qpubio/qpub-server/internal/domain/messaging/connection"
	"github.com/qpubio/qpub-server/internal/domain/messaging/delivery"
	domainTelemetry "github.com/qpubio/qpub-server/internal/domain/messaging/telemetry"
	"github.com/qpubio/qpub-server/internal/domain/messaging/subscription"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

const (
	defaultSubscriptionBuffer = 256
	defaultEgressCapacity     = 256
)

// Service owns per-connection egress pipelines and client subscription state.
type Service struct {
	logger            logger.Service
	instanceID        id.ULID
	connectionService connection.Service
	telemetryService  domainTelemetry.Service
	pipelines         *egress.Registry

	mu                sync.RWMutex
	clientConn        map[id.ULID]id.ULID
	subscriptionCache map[id.ULID]*subscription.Subscription
	clientMutexes     map[id.ULID]*sync.Mutex
}

// NewService creates a session registry service.
func NewService(
	logger logger.Service,
	instanceID id.ULID,
	connectionService connection.Service,
	telemetryService domainTelemetry.Service,
) *Service {
	return &Service{
		logger:            logger,
		instanceID:        instanceID,
		connectionService: connectionService,
		telemetryService:  telemetryService,
		pipelines:         egress.NewRegistry(),
		clientConn:        make(map[id.ULID]id.ULID),
		subscriptionCache: make(map[id.ULID]*subscription.Subscription),
		clientMutexes:     make(map[id.ULID]*sync.Mutex),
	}
}

// RegisterConnection starts an outbound pipeline for a connection.
func (s *Service) RegisterConnection(connID id.ULID, projectID id.Int, clientID id.ULID) {
	pipeline := egress.NewPipeline(egress.PipelineParams{
		ConnID:            connID,
		ProjectID:         projectID,
		InstanceID:        s.instanceID,
		ConnectionService: s.connectionService,
		TelemetryService:  s.telemetryService,
		Logger:            s.logger,
		Capacity:          defaultEgressCapacity,
		OnSlowConsumer: func(connID id.ULID) {
			if err := s.connectionService.Close(connID); err != nil {
				s.logger.Warn(log.MessagingConnection,
					"Failed to close slow consumer connection connectionID=%s error=%v",
					connID, err)
			}
		},
	})
	pipeline.Start()
	s.pipelines.Register(connID, pipeline)

	s.mu.Lock()
	s.clientConn[clientID] = connID
	s.mu.Unlock()

	s.logger.Debug(log.MessagingConnection,
		"Session registered connection egress pipeline connectionID=%s clientID=%s projectID=%d",
		connID, clientID, projectID)
}

// UnregisterConnection stops the egress pipeline and clears client session state.
func (s *Service) UnregisterConnection(connID id.ULID, clientID id.ULID) {
	s.pipelines.Unregister(connID)

	s.mu.Lock()
	delete(s.clientConn, clientID)
	delete(s.subscriptionCache, clientID)
	delete(s.clientMutexes, clientID)
	s.mu.Unlock()

	s.logger.Debug(log.MessagingConnection,
		"Session unregistered connection connectionID=%s clientID=%s",
		connID, clientID)
}

// GetOrCreateSubscription returns the reusable subscription for a client.
func (s *Service) GetOrCreateSubscription(clientID id.ULID) *subscription.Subscription {
	mu := s.clientMutex(clientID)
	mu.Lock()
	defer mu.Unlock()

	s.mu.RLock()
	sub, ok := s.subscriptionCache[clientID]
	s.mu.RUnlock()

	if ok && sub != nil && !sub.IsClosed() {
		return sub
	}

	sub = subscription.New(clientID, defaultSubscriptionBuffer)

	s.mu.Lock()
	s.subscriptionCache[clientID] = sub
	s.mu.Unlock()

	s.logger.Debug(log.MessagingSubscription,
		"Created session subscription clientID=%s subscriptionID=%s",
		clientID, sub.ID())

	return sub
}

// Deliver enqueues an outbound payload for the client's connection.
func (s *Service) Deliver(clientID id.ULID, payload []byte) error {
	s.mu.RLock()
	connID, ok := s.clientConn[clientID]
	s.mu.RUnlock()

	if !ok {
		return connection.ErrConnectionNotFound
	}

	return s.pipelines.Enqueue(connID, payload)
}

func (s *Service) clientMutex(clientID id.ULID) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()

	if mu, ok := s.clientMutexes[clientID]; ok {
		return mu
	}

	mu := &sync.Mutex{}
	s.clientMutexes[clientID] = mu
	return mu
}

var _ delivery.Deliverer = (*Service)(nil)
