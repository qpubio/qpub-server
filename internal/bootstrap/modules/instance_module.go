package modules

import (
	"reflect"
	"time"

	instanceService "github.com/qpubio/qpub-server/internal/application/service/cluster/instance"
	"github.com/qpubio/qpub-server/internal/bootstrap/container"
	"github.com/qpubio/qpub-server/internal/config"
	"github.com/qpubio/qpub-server/internal/domain/cluster/instance"
	"github.com/qpubio/qpub-server/internal/domain/project/stat/realtime"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	instanceRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/cluster/instance"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"gorm.io/gorm"
)

type InstanceServiceManager struct {
	instance.Service
	config     *config.Config
	logger     logger.Service
	instanceID id.ULID
	stopChan   chan struct{}
	stopped    bool
}

func NewInstanceServiceManager(
	config *config.Config,
	logger logger.Service,
	instanceID id.ULID,
	service instance.Service,
) *InstanceServiceManager {
	return &InstanceServiceManager{
		Service:    service,
		config:     config,
		logger:     logger,
		instanceID: instanceID,
		stopChan:   make(chan struct{}),
	}
}

func (s *InstanceServiceManager) Initialize() error {
	s.logger.Info(log.Instance, "Initializing instance service manager")
	if err := s.Register(); err != nil {
		return err
	}
	s.startHeartbeat()
	s.logger.Info(log.Instance, "Instance service manager initialized successfully")
	return nil
}

func (s *InstanceServiceManager) Shutdown() error {
	if s.stopped {
		return nil
	}
	s.logger.Info(log.Instance, "Shutting down instance service manager")
	close(s.stopChan)
	s.stopped = true
	if err := s.Deregister(); err != nil {
		s.logger.Error(log.Instance, "Failed to deregister instance during shutdown: %v", err)
		return err
	}
	s.logger.Info(log.Instance, "Instance service manager shutdown complete")
	return nil
}

func (s *InstanceServiceManager) startHeartbeat() {
	heartbeatInterval := time.Duration(s.config.Infrastructure.Instance.Heartbeat.Interval) * time.Second
	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		s.logger.Info(log.Instance, "Starting instance heartbeat with interval %v", heartbeatInterval)
		for {
			select {
			case <-ticker.C:
				if err := s.Heartbeat(); err != nil {
					s.logger.Warn(log.Instance, "Failed to update instance heartbeat: %v", err)
				}
			case <-s.stopChan:
				s.logger.Info(log.Instance, "Stopping instance heartbeat")
				return
			}
		}
	}()
}

type InstanceModule struct {
	container.ModuleBase
}

func NewInstanceModule() *InstanceModule {
	return &InstanceModule{
		ModuleBase: container.NewModule("instance", 5),
	}
}

func (m *InstanceModule) Register(c *container.Container) error {
	c.Register(
		reflect.TypeOf((*instance.Repository)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			db, err := container.GetTyped[*gorm.DB](c)
			if err != nil {
				return nil, err
			}
			return instanceRepo.NewRepository(db), nil
		},
	)

	c.Register(
		reflect.TypeOf((*instance.Service)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			cfg, err := container.GetTyped[*config.Config](c)
			if err != nil {
				return nil, err
			}
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}
			instanceID, err := container.GetTyped[id.ULID](c)
			if err != nil {
				return nil, err
			}
			repository, err := container.GetTyped[instance.Repository](c)
			if err != nil {
				return nil, err
			}

			var statService realtime.Service
			statService, err = container.GetTyped[realtime.Service](c)
			if err != nil {
				logger.Warn(log.Instance, "Realtime stat service not available yet; instance tracking limited")
			}

			instanceSvc := instanceService.NewService(cfg, logger, instanceID, repository, statService)
			return NewInstanceServiceManager(cfg, logger, instanceID, instanceSvc), nil
		},
	)

	if logger, err := container.GetTyped[logger.Service](c); err == nil {
		logger.Info(log.App, "Instance module registered successfully")
	}
	return nil
}
