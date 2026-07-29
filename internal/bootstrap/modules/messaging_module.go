package modules

import (
	"context"
	"fmt"
	"reflect"
	"time"

	brokerSvc "github.com/qpubio/qpub-server/internal/application/service/messaging/broker"
	channelService "github.com/qpubio/qpub-server/internal/application/service/messaging/channel"
	"github.com/qpubio/qpub-server/internal/application/service/messaging/channel/lifecycle"
	clientService "github.com/qpubio/qpub-server/internal/application/service/messaging/client"
	connectionService "github.com/qpubio/qpub-server/internal/application/service/messaging/connection"
	eventService "github.com/qpubio/qpub-server/internal/application/service/messaging/event"
	publicationService "github.com/qpubio/qpub-server/internal/application/service/messaging/publication"
	routerService "github.com/qpubio/qpub-server/internal/application/service/messaging/router"
	backpressureApp "github.com/qpubio/qpub-server/internal/application/service/messaging/backpressure"
	sessionService "github.com/qpubio/qpub-server/internal/application/service/messaging/session"
	telemetryService "github.com/qpubio/qpub-server/internal/application/service/messaging/telemetry"
	telemetrySnapshot "github.com/qpubio/qpub-server/internal/application/service/messaging/telemetry/snapshot"
	subscriptionService "github.com/qpubio/qpub-server/internal/application/service/messaging/subscription"
	"github.com/qpubio/qpub-server/internal/bootstrap/container"
	"github.com/qpubio/qpub-server/internal/config/messaging"
	"github.com/qpubio/qpub-server/internal/domain/messaging/broker"
	"github.com/qpubio/qpub-server/internal/domain/messaging/channel"
	"github.com/qpubio/qpub-server/internal/domain/messaging/client"
	"github.com/qpubio/qpub-server/internal/domain/messaging/connection"
	"github.com/qpubio/qpub-server/internal/domain/messaging/delivery"
	"github.com/qpubio/qpub-server/internal/domain/messaging/event"
	"github.com/qpubio/qpub-server/internal/domain/messaging/backpressure"
	"github.com/qpubio/qpub-server/internal/domain/messaging/publication"
	domainRouter "github.com/qpubio/qpub-server/internal/domain/messaging/router"
	domainTelemetry "github.com/qpubio/qpub-server/internal/domain/messaging/telemetry"
	"github.com/qpubio/qpub-server/internal/domain/messaging/subscription"
	"github.com/qpubio/qpub-server/internal/domain/project/stat/realtime"
	"github.com/qpubio/qpub-server/internal/domain/tenant"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	brokerRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/messaging/broker"
	channelRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/messaging/channel"
	clientRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/messaging/client"
	connectionRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/messaging/connection"
	telemetryRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/messaging/telemetry"
	subscriptionRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/messaging/subscription"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"github.com/caarlos0/env/v11"
	"github.com/nats-io/nats.go"
)

// EventBusLifecycle wraps the event bus with lifecycle methods
type EventBusLifecycle struct {
	event.Service
	logger logger.Service
}

func (e *EventBusLifecycle) Initialize() error {
	e.logger.Info(log.App, "Event bus service initialized")
	return nil
}

func (e *EventBusLifecycle) Shutdown() error {
	e.logger.Info(log.App, "Shutting down event bus service")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return e.Service.Close(shutdownCtx)
}

// ChannelLifecycleManagerLifecycle wraps the channel lifecycle manager
type ChannelLifecycleManagerLifecycle struct {
	*lifecycle.Manager
	logger logger.Service
}

func (c *ChannelLifecycleManagerLifecycle) Initialize() error {
	c.logger.Info(log.MessagingChannel, "Channel lifecycle manager initialized")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.Manager.Start(ctx)
}

func (c *ChannelLifecycleManagerLifecycle) Shutdown() error {
	c.logger.Info(log.MessagingChannel, "Shutting down channel lifecycle manager")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.Manager.Stop(shutdownCtx)
}

// BrokerServiceLifecycle wraps a broker service with lifecycle methods
type BrokerServiceLifecycle struct {
	broker.Service
	logger logger.Service
}

// Initialize implements the container.Lifecycle interface
func (s *BrokerServiceLifecycle) Initialize() error {
	s.logger.Info(log.MessagingBroker, "Broker service initialized")
	return nil
}

// Shutdown implements the container.Lifecycle interface
func (s *BrokerServiceLifecycle) Shutdown() error {
	s.logger.Info(log.MessagingBroker, "Shutting down broker service")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.Service.Shutdown(shutdownCtx)
}

// TelemetrySnapshotServiceLifecycle wraps the telemetry snapshot service
type TelemetrySnapshotServiceLifecycle struct {
	service *telemetrySnapshot.Service
	logger  logger.Service
}

func (s *TelemetrySnapshotServiceLifecycle) Initialize() error {
	s.logger.Info(log.Stats, "Telemetry snapshot service initialized")
	s.service.Start()
	return nil
}

func (s *TelemetrySnapshotServiceLifecycle) Shutdown() error {
	s.logger.Info(log.Stats, "Shutting down telemetry snapshot service")
	s.service.Stop()
	return nil
}

// MessagingModule provides registration for enhanced messaging-related services
type MessagingModule struct {
	container.ModuleBase
}

// NewMessagingModule creates a new enhanced messaging module
func NewMessagingModule() *MessagingModule {
	return &MessagingModule{
		ModuleBase: container.NewModule("messaging", 40), // After project (30)
	}
}

// Register registers all enhanced messaging-related services
func (m *MessagingModule) Register(c *container.Container) error {
	// Register messaging configuration
	c.Register(
		reflect.TypeOf((*messaging.ChannelConfig)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			config := &messaging.ChannelConfig{}
			if err := env.Parse(config); err != nil {
				return nil, fmt.Errorf("failed to parse messaging channel config: %w", err)
			}
			return config, nil
		},
	)

	// Register event bus service
	c.Register(
		reflect.TypeOf((*event.Service)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			return eventService.NewService(logger), nil
		},
	)

	// Register event bus lifecycle
	c.RegisterDescriptor(
		container.NewDescriptor(
			reflect.TypeOf((*EventBusLifecycle)(nil)),
			func(c *container.Container) (interface{}, error) {
				logger, err := container.GetTyped[logger.Service](c)
				if err != nil {
					return nil, err
				}

				eventBus, err := container.GetTyped[event.Service](c)
				if err != nil {
					return nil, err
				}

				return &EventBusLifecycle{
					Service: eventBus,
					logger:  logger,
				}, nil
			},
		),
	)

	// Register telemetry repository (in-memory)
	c.Register(
		reflect.TypeOf((*domainTelemetry.Repository)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			return telemetryRepo.NewRepository(logger), nil
		},
	)

	// Register telemetry service
	c.Register(
		reflect.TypeOf((*domainTelemetry.Service)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			instanceID, err := container.GetTyped[id.ULID](c)
			if err != nil {
				return nil, err
			}

			repository, err := container.GetTyped[domainTelemetry.Repository](c)
			if err != nil {
				return nil, err
			}

			return telemetryService.NewService(logger, instanceID, repository), nil
		},
	)

	// Register telemetry snapshot service
	c.Register(
		reflect.TypeOf((*telemetrySnapshot.Service)(nil)),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			instanceID, err := container.GetTyped[id.ULID](c)
			if err != nil {
				return nil, err
			}

			subscriptionRepo, err := container.GetTyped[subscription.Repository](c)
			if err != nil {
				return nil, err
			}

			channelRepo, err := container.GetTyped[channel.Repository](c)
			if err != nil {
				return nil, err
			}

			connectionRepo, err := container.GetTyped[connection.Repository](c)
			if err != nil {
				return nil, err
			}

			telemetryRepository, err := container.GetTyped[domainTelemetry.Repository](c)
			if err != nil {
				return nil, err
			}

			realtimeService, err := container.GetTyped[realtime.Service](c)
			if err != nil {
				return nil, err
			}

			return telemetrySnapshot.NewService(
				logger,
				instanceID,
				subscriptionRepo,
				channelRepo,
				connectionRepo,
				telemetryRepository,
				realtimeService,
			), nil
		},
	)

	// Register telemetry snapshot service lifecycle
	c.RegisterDescriptor(
		container.NewDescriptor(
			reflect.TypeOf((*TelemetrySnapshotServiceLifecycle)(nil)),
			func(c *container.Container) (interface{}, error) {
				logger, err := container.GetTyped[logger.Service](c)
				if err != nil {
					return nil, err
				}

				svc, err := container.GetTyped[*telemetrySnapshot.Service](c)
				if err != nil {
					return nil, err
				}

				return &TelemetrySnapshotServiceLifecycle{
					service: svc,
					logger:  logger,
				}, nil
			},
		),
	)

	// Register main broker service (NATS-based)
	c.Register(
		reflect.TypeOf((*broker.Service)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			instanceID, err := container.GetTyped[id.ULID](c)
			if err != nil {
				return nil, err
			}

			natsConn, err := container.GetTyped[*nats.Conn](c)
			if err != nil {
				return nil, err
			}

			brokerRepository := brokerRepo.NewRepository(natsConn, logger)
			brokerService := brokerSvc.NewService(brokerRepository, instanceID, logger)

			return brokerService, nil
		},
	)

	// Register broker service lifecycle
	c.RegisterDescriptor(
		container.NewDescriptor(
			reflect.TypeOf((*BrokerServiceLifecycle)(nil)),
			func(c *container.Container) (interface{}, error) {
				logger, err := container.GetTyped[logger.Service](c)
				if err != nil {
					return nil, err
				}

				brokerService, err := container.GetTyped[broker.Service](c)
				if err != nil {
					return nil, err
				}

				return &BrokerServiceLifecycle{
					Service: brokerService,
					logger:  logger,
				}, nil
			},
		),
	)

	// Register connection repository
	c.Register(
		reflect.TypeOf((*connection.Repository)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			return connectionRepo.NewRepository(logger), nil
		},
	)

	// Register connection service
	c.Register(
		reflect.TypeOf((*connection.Service)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			instanceID, err := container.GetTyped[id.ULID](c)
			if err != nil {
				return nil, err
			}

			repository, err := container.GetTyped[connection.Repository](c)
			if err != nil {
				return nil, err
			}

			return connectionService.NewService(logger, instanceID, repository), nil
		},
	)

	// Register client repository
	c.Register(
		reflect.TypeOf((*client.Repository)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			return clientRepo.NewRepository(logger), nil
		},
	)

	// Register client service
	c.Register(
		reflect.TypeOf((*client.Service)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			instanceID, err := container.GetTyped[id.ULID](c)
			if err != nil {
				return nil, err
			}

			repository, err := container.GetTyped[client.Repository](c)
			if err != nil {
				return nil, err
			}

			connectionService, err := container.GetTyped[connection.Service](c)
			if err != nil {
				return nil, err
			}

			return clientService.NewService(logger, instanceID, repository, connectionService), nil
		},
	)

	// Register channel repository
	c.Register(
		reflect.TypeOf((*channel.Repository)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			return channelRepo.NewRepository(logger), nil
		},
	)

	// Register channel service
	c.Register(
		reflect.TypeOf((*channel.Service)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			instanceID, err := container.GetTyped[id.ULID](c)
			if err != nil {
				return nil, err
			}

			repository, err := container.GetTyped[channel.Repository](c)
			if err != nil {
				return nil, err
			}

			brokerService, err := container.GetTyped[broker.Service](c)
			if err != nil {
				return nil, err
			}

		eventBus, err := container.GetTyped[event.Service](c)
		if err != nil {
			return nil, err
		}

		return channelService.NewService(logger, instanceID, repository, brokerService, eventBus), nil
		},
	)

	// Register channel lifecycle manager
	c.Register(
		reflect.TypeOf((*lifecycle.Manager)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			channelService, err := container.GetTyped[channel.Service](c)
			if err != nil {
				return nil, err
			}

			eventBus, err := container.GetTyped[event.Service](c)
			if err != nil {
				return nil, err
			}

			channelConfig, err := container.GetTyped[messaging.ChannelConfig](c)
			if err != nil {
				return nil, err
			}

			return lifecycle.NewManager(
				logger,
				channelService,
				eventBus,
				channelConfig.GetCleanupDelay(),
			), nil
		},
	)

	// Register channel lifecycle manager lifecycle
	c.RegisterDescriptor(
		container.NewDescriptor(
			reflect.TypeOf((*ChannelLifecycleManagerLifecycle)(nil)),
			func(c *container.Container) (interface{}, error) {
				logger, err := container.GetTyped[logger.Service](c)
				if err != nil {
					return nil, err
				}

				manager, err := container.GetTyped[*lifecycle.Manager](c)
				if err != nil {
					return nil, err
				}

				return &ChannelLifecycleManagerLifecycle{
					Manager: manager,
					logger:  logger,
				}, nil
			},
		),
	)

	// Register subscription repository
	c.Register(
		reflect.TypeOf((*subscription.Repository)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			instanceID, err := container.GetTyped[id.ULID](c)
			if err != nil {
				return nil, err
			}

			return subscriptionRepo.NewRepository(logger, instanceID), nil
		},
	)

	// Register session service (connection registry + egress delivery)
	c.Register(
		reflect.TypeOf((*sessionService.Service)(nil)),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			instanceID, err := container.GetTyped[id.ULID](c)
			if err != nil {
				return nil, err
			}

			connSvc, err := container.GetTyped[connection.Service](c)
			if err != nil {
				return nil, err
			}

			telemetrySvc, err := container.GetTyped[domainTelemetry.Service](c)
			if err != nil {
				return nil, err
			}

			return sessionService.NewService(logger, instanceID, connSvc, telemetrySvc), nil
		},
	)

	c.Register(
		reflect.TypeOf((*delivery.Deliverer)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			return container.GetTyped[*sessionService.Service](c)
		},
	)

	// Register backpressure limits + gatekeeper
	c.Register(
		reflect.TypeOf((*backpressure.LimitsProvider)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			tenantService, err := container.GetTyped[tenant.Service](c)
			if err != nil {
				return nil, err
			}

			logger.Info(log.App, "Using store-backed message rate limits")
			return backpressureApp.NewStoreLimitsProvider(tenantService), nil
		},
	)

	c.Register(
		reflect.TypeOf((*backpressure.Gatekeeper)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			limitsProvider, err := container.GetTyped[backpressure.LimitsProvider](c)
			if err != nil {
				return nil, err
			}

			return backpressureApp.NewGatekeeperService(logger, limitsProvider), nil
		},
	)

	// Register message router
	c.Register(
		reflect.TypeOf((*domainRouter.Service)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			instanceID, err := container.GetTyped[id.ULID](c)
			if err != nil {
				return nil, err
			}

			brokerService, err := container.GetTyped[broker.Service](c)
			if err != nil {
				return nil, err
			}

			channelService, err := container.GetTyped[channel.Service](c)
			if err != nil {
				return nil, err
			}

			subscriptionRepo, err := container.GetTyped[subscription.Repository](c)
			if err != nil {
				return nil, err
			}

			deliverer, err := container.GetTyped[delivery.Deliverer](c)
			if err != nil {
				return nil, err
			}

			telemetrySvc, err := container.GetTyped[domainTelemetry.Service](c)
			if err != nil {
				return nil, err
			}

			gatekeeper, err := container.GetTyped[backpressure.Gatekeeper](c)
			if err != nil {
				return nil, err
			}

			return routerService.NewService(
				logger,
				instanceID,
				brokerService,
				channelService,
				subscriptionRepo,
				deliverer,
				telemetrySvc,
				gatekeeper,
			), nil
		},
	)

	// Register enhanced subscription service
	c.Register(
		reflect.TypeOf((*subscription.Service)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			instanceID, err := container.GetTyped[id.ULID](c)
			if err != nil {
				return nil, err
			}

			repository, err := container.GetTyped[subscription.Repository](c)
			if err != nil {
				return nil, err
			}

			channelService, err := container.GetTyped[channel.Service](c)
			if err != nil {
				return nil, err
			}

			router, err := container.GetTyped[domainRouter.Service](c)
			if err != nil {
				return nil, err
			}

			deliverer, err := container.GetTyped[delivery.Deliverer](c)
			if err != nil {
				return nil, err
			}

			eventBus, err := container.GetTyped[event.Service](c)
			if err != nil {
				return nil, err
			}

			return subscriptionService.NewService(
				logger,
				instanceID,
				repository,
				channelService,
				router,
				deliverer,
				eventBus,
			), nil
		},
	)

	// Register publication service
	c.Register(
		reflect.TypeOf((*publication.Service)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}

			clientService, err := container.GetTyped[client.Service](c)
			if err != nil {
				return nil, err
			}

			router, err := container.GetTyped[domainRouter.Service](c)
			if err != nil {
				return nil, err
			}

			return publicationService.NewService(logger, router, clientService), nil
		},
	)

	logger, err := container.GetTyped[logger.Service](c)
	if err == nil {
		logger.Info(log.App, "Enhanced messaging module registered successfully")
	}

	return nil
}
