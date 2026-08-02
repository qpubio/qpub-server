package bootstrap

import (
	"reflect"

	"github.com/qpubio/qpub-server/internal/bootstrap/container"
	"github.com/qpubio/qpub-server/internal/bootstrap/modules"
	"github.com/qpubio/qpub-server/internal/domain/cluster/instance"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/infrastructure/nats"
	"github.com/qpubio/qpub-server/internal/infrastructure/redis"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

func (a *App) setupContainer() error {
	a.logger.Info(log.App, "Setting up dependency container...")

	cont := container.New()
	cont.RegisterInstance(reflect.TypeOf((*logger.Service)(nil)).Elem(), a.logger)
	cont.RegisterInstance(reflect.TypeOf(a.config), a.config)
	cont.RegisterInstance(reflect.TypeOf(a.db), a.db)
	cont.RegisterInstance(reflect.TypeOf(a.redis), a.redis)
	cont.RegisterInstance(reflect.TypeOf((*redis.Service)(nil)).Elem(), a.redis)
	cont.RegisterInstance(reflect.TypeOf(a.nats), a.nats)
	cont.RegisterInstance(reflect.TypeOf((*nats.Service)(nil)).Elem(), a.nats)
	cont.RegisterInstance(reflect.TypeOf(a.instanceID), a.instanceID)
	cont.RegisterInstance(reflect.TypeOf(a.apikeyParser), a.apikeyParser)

	a.container = cont
	a.cleanup.Register(func() error {
		a.logger.Info(log.App, "Shutting down container services...")
		return a.container.Shutdown()
	})

	a.logger.Info(log.App, "Dependency container setup complete")
	return nil
}

func (a *App) registerServiceModules() error {
	a.logger.Info(log.App, "Registering service modules...")

	moduleList := []container.Module{
		modules.NewInfrastructureModule(), // 1
		modules.NewInstanceModule(),       // 5
		modules.NewAPIKeyModule(),         // 31
		modules.NewProjectStatModule(),    // 34
		modules.NewTenantModule(),         // 39
		modules.NewMessagingModule(),      // 40
		modules.NewQueueModule(),          // 41
		modules.NewAuthModule(),           // 60
	}

	if err := container.RegisterModules(a.container, moduleList...); err != nil {
		return err
	}

	a.logger.Info(log.App, "Service modules registered successfully")
	return nil
}

func (a *App) initializeLifecycleServices() error {
	a.logger.Info(log.App, "Initializing lifecycle services...")

	// Resolve instance service so register/heartbeat Lifecycle hooks run.
	if _, err := container.GetTyped[instance.Service](a.container); err != nil {
		return err
	}

	lifecycleTypes := []reflect.Type{
		reflect.TypeOf((*modules.EventBusLifecycle)(nil)),
		reflect.TypeOf((*modules.TelemetrySnapshotServiceLifecycle)(nil)),
		reflect.TypeOf((*modules.QueueRuntimeLifecycle)(nil)),
	}

	for _, lifecycleType := range lifecycleTypes {
		if _, err := a.container.Get(lifecycleType); err != nil {
			return err
		}
	}

	a.logger.Info(log.App, "Lifecycle services initialized successfully")
	return nil
}
