package modules

import (
	"context"
	"reflect"
	"time"

	queueBackpressureApp "github.com/qpubio/qpub-server/internal/application/service/queue/backpressure"
	"github.com/qpubio/qpub-server/internal/application/service/queue/dispatch/webhook"
	queueJobApp "github.com/qpubio/qpub-server/internal/application/service/queue/job"
	"github.com/qpubio/qpub-server/internal/application/service/queue/platform"
	queueApp "github.com/qpubio/qpub-server/internal/application/service/queue/queue"
	queueRouterApp "github.com/qpubio/qpub-server/internal/application/service/queue/router"
	queueRuntimeApp "github.com/qpubio/qpub-server/internal/application/service/queue/runtime"
	queueScheduleApp "github.com/qpubio/qpub-server/internal/application/service/queue/schedule"
	queueTelemetryApp "github.com/qpubio/qpub-server/internal/application/service/queue/telemetry"
	queueWorkerApp "github.com/qpubio/qpub-server/internal/application/service/queue/worker"
	"github.com/qpubio/qpub-server/internal/bootstrap/container"
	"github.com/qpubio/qpub-server/internal/domain/messaging/publication"
	msgTelemetry "github.com/qpubio/qpub-server/internal/domain/messaging/telemetry"
	logBroadcast "github.com/qpubio/qpub-server/internal/domain/project/log/broadcast"
	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
	domainRouter "github.com/qpubio/qpub-server/internal/domain/queue/router"
	domainRuntime "github.com/qpubio/qpub-server/internal/domain/queue/runtime"
	domainTelemetry "github.com/qpubio/qpub-server/internal/domain/queue/telemetry"
	domainWorker "github.com/qpubio/qpub-server/internal/domain/queue/worker"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/infrastructure/nats"
	"github.com/qpubio/qpub-server/internal/infrastructure/redis"
	brokerRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/queue/broker"
	jobRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/queue/job"
	queueRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/queue/queue"
	telemetryRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/queue/telemetry"
	workerRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/queue/worker"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"gorm.io/gorm"
)

type QueueModule struct {
	container.ModuleBase
}

func NewQueueModule() *QueueModule {
	return &QueueModule{
		ModuleBase: container.NewModule("queue", 41), // After messaging (40) for telemetry + _logs
	}
}

type QueueRuntimeLifecycle struct {
	runtime  domainRuntime.Service
	schedule *queueScheduleApp.Service
	logger   logger.Service
}

func (l *QueueRuntimeLifecycle) Initialize() error {
	l.logger.Info(log.Queue, "Starting queue runtime")
	ctx := context.Background()
	if err := l.schedule.Start(ctx); err != nil {
		return err
	}
	return l.runtime.Start(ctx)
}

func (l *QueueRuntimeLifecycle) Shutdown() error {
	l.logger.Info(log.Queue, "Stopping queue runtime")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = l.runtime.Stop(ctx)
	return l.schedule.Stop(ctx)
}

func (m *QueueModule) Register(c *container.Container) error {
	c.Register(reflect.TypeOf((*domainQueue.Repository)(nil)).Elem(), func(c *container.Container) (interface{}, error) {
		db, err := container.GetTyped[*gorm.DB](c)
		if err != nil {
			return nil, err
		}
		return queueRepo.NewRepository(db), nil
	})

	c.Register(reflect.TypeOf((*domainJob.Repository)(nil)).Elem(), func(c *container.Container) (interface{}, error) {
		db, err := container.GetTyped[*gorm.DB](c)
		if err != nil {
			return nil, err
		}
		return jobRepo.NewRepository(db), nil
	})

	c.Register(reflect.TypeOf((*domainWorker.Repository)(nil)).Elem(), func(c *container.Container) (interface{}, error) {
		db, err := container.GetTyped[*gorm.DB](c)
		if err != nil {
			return nil, err
		}
		return workerRepo.NewRepository(db), nil
	})

	c.Register(reflect.TypeOf((*domainTelemetry.Repository)(nil)).Elem(), func(c *container.Container) (interface{}, error) {
		return telemetryRepo.NewRepository(), nil
	})

	c.Register(reflect.TypeOf((*domainTelemetry.Service)(nil)).Elem(), func(c *container.Container) (interface{}, error) {
		repo, err := container.GetTyped[domainTelemetry.Repository](c)
		if err != nil {
			return nil, err
		}
		return queueTelemetryApp.NewService(repo), nil
	})

	c.Register(reflect.TypeOf((*domainQueue.Service)(nil)).Elem(), func(c *container.Container) (interface{}, error) {
		repo, err := container.GetTyped[domainQueue.Repository](c)
		if err != nil {
			return nil, err
		}
		logger, err := container.GetTyped[logger.Service](c)
		if err != nil {
			return nil, err
		}
		return queueApp.NewService(repo, logger), nil
	})

	c.Register(reflect.TypeOf((*logBroadcast.Service)(nil)).Elem(), func(c *container.Container) (interface{}, error) {
		loggerSvc, err := container.GetTyped[logger.Service](c)
		if err != nil {
			return nil, err
		}
		publicationService, err := container.GetTyped[publication.Service](c)
		if err != nil {
			return nil, err
		}
		return NewProjectLogBroadcastModule(loggerSvc, publicationService), nil
	})

	c.Register(reflect.TypeOf((*domainJob.Service)(nil)).Elem(), func(c *container.Container) (interface{}, error) {
		repo, err := container.GetTyped[domainJob.Repository](c)
		if err != nil {
			return nil, err
		}
		logger, err := container.GetTyped[logger.Service](c)
		if err != nil {
			return nil, err
		}
		broadcaster, err := container.GetTyped[logBroadcast.Service](c)
		if err != nil {
			return nil, err
		}
		return queueJobApp.NewService(repo, logger, broadcaster), nil
	})

	c.Register(reflect.TypeOf((*domainWorker.Service)(nil)).Elem(), func(c *container.Container) (interface{}, error) {
		repo, err := container.GetTyped[domainWorker.Repository](c)
		if err != nil {
			return nil, err
		}
		jobRepository, err := container.GetTyped[domainJob.Repository](c)
		if err != nil {
			return nil, err
		}
		logger, err := container.GetTyped[logger.Service](c)
		if err != nil {
			return nil, err
		}
		broadcaster, err := container.GetTyped[logBroadcast.Service](c)
		if err != nil {
			return nil, err
		}
		return queueWorkerApp.NewService(repo, jobRepository, logger, broadcaster), nil
	})

	c.Register(reflect.TypeOf((*domainRouter.Service)(nil)).Elem(), func(c *container.Container) (interface{}, error) {
		jobRepository, err := container.GetTyped[domainJob.Repository](c)
		if err != nil {
			return nil, err
		}
		queueRepository, err := container.GetTyped[domainQueue.Repository](c)
		if err != nil {
			return nil, err
		}
		natsService, err := container.GetTyped[nats.Service](c)
		if err != nil {
			return nil, err
		}
		logger, err := container.GetTyped[logger.Service](c)
		if err != nil {
			return nil, err
		}
		broker, err := brokerRepo.NewRepository(natsService, logger)
		if err != nil {
			return nil, err
		}
		gatekeeper := queueBackpressureApp.NewService()
		telemetry, err := container.GetTyped[domainTelemetry.Service](c)
		if err != nil {
			return nil, err
		}
		messagingTelemetry, err := container.GetTyped[msgTelemetry.Service](c)
		if err != nil {
			return nil, err
		}
		broadcaster, err := container.GetTyped[logBroadcast.Service](c)
		if err != nil {
			return nil, err
		}
		instanceID, err := container.GetTyped[id.ULID](c)
		if err != nil {
			return nil, err
		}
		queueService, err := container.GetTyped[domainQueue.Service](c)
		if err != nil {
			return nil, err
		}
		workerService, err := container.GetTyped[domainWorker.Service](c)
		if err != nil {
			return nil, err
		}
		return queueRouterApp.NewService(
			jobRepository,
			queueRepository,
			broker,
			gatekeeper,
			telemetry,
			messagingTelemetry,
			broadcaster,
			workerService,
			instanceID,
			queueService,
			logger,
		), nil
	})

	c.Register(reflect.TypeOf((*platform.Registry)(nil)), func(c *container.Container) (interface{}, error) {
		return platform.NewRegistry(), nil
	})

	c.Register(reflect.TypeOf((*webhook.Service)(nil)), func(c *container.Container) (interface{}, error) {
		return webhook.NewService(), nil
	})

	c.Register(reflect.TypeOf((*domainRuntime.Service)(nil)).Elem(), func(c *container.Container) (interface{}, error) {
		router, err := container.GetTyped[domainRouter.Service](c)
		if err != nil {
			return nil, err
		}
		jobRepository, err := container.GetTyped[domainJob.Repository](c)
		if err != nil {
			return nil, err
		}
		queueRepository, err := container.GetTyped[domainQueue.Repository](c)
		if err != nil {
			return nil, err
		}
		webhookSvc, err := container.GetTyped[*webhook.Service](c)
		if err != nil {
			return nil, err
		}
		platformReg, err := container.GetTyped[*platform.Registry](c)
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
		return queueRuntimeApp.NewService(router, jobRepository, queueRepository, webhookSvc, platformReg, logger, instanceID), nil
	})

	c.Register(reflect.TypeOf((*queueScheduleApp.Service)(nil)), func(c *container.Container) (interface{}, error) {
		redisService, err := container.GetTyped[redis.Service](c)
		if err != nil {
			return nil, err
		}
		router, err := container.GetTyped[domainRouter.Service](c)
		if err != nil {
			return nil, err
		}
		platformReg, err := container.GetTyped[*platform.Registry](c)
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
		return queueScheduleApp.NewService(redisService, router, platformReg, logger, instanceID), nil
	})

	c.Register(reflect.TypeOf((*QueueRuntimeLifecycle)(nil)), func(c *container.Container) (interface{}, error) {
		runtime, err := container.GetTyped[domainRuntime.Service](c)
		if err != nil {
			return nil, err
		}
		schedule, err := container.GetTyped[*queueScheduleApp.Service](c)
		if err != nil {
			return nil, err
		}
		logger, err := container.GetTyped[logger.Service](c)
		if err != nil {
			return nil, err
		}
		return &QueueRuntimeLifecycle{runtime: runtime, schedule: schedule, logger: logger}, nil
	})

	logger, err := container.GetTyped[logger.Service](c)
	if err == nil {
		logger.Info(log.App, "Queue module registered successfully")
	}
	return nil
}

