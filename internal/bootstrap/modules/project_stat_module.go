package modules

import (
	"reflect"

	projectStatRealtimeService "github.com/qpubio/qpub-server/internal/application/service/project/stat/realtime"
	"github.com/qpubio/qpub-server/internal/bootstrap/container"
	"github.com/qpubio/qpub-server/internal/domain/project/stat/realtime"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	projectStatRealtimeRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/project/stat/realtime"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"github.com/go-redis/redis"
)

// ProjectStatModule registers Redis-backed realtime stats used by messaging telemetry.
type ProjectStatModule struct {
	container.ModuleBase
}

func NewProjectStatModule() *ProjectStatModule {
	return &ProjectStatModule{
		ModuleBase: container.NewModule("project-stat", 34),
	}
}

func (m *ProjectStatModule) Register(c *container.Container) error {
	c.Register(
		reflect.TypeOf((*realtime.Repository)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			redisClient, err := container.GetTyped[*redis.Client](c)
			if err != nil {
				return nil, err
			}
			return projectStatRealtimeRepo.NewRepository(redisClient), nil
		},
	)

	c.Register(
		reflect.TypeOf((*realtime.Service)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			repository, err := container.GetTyped[realtime.Repository](c)
			if err != nil {
				return nil, err
			}
			return projectStatRealtimeService.NewService(repository), nil
		},
	)

	if logger, err := container.GetTyped[logger.Service](c); err == nil {
		logger.Info(log.App, "Project stat module registered successfully")
	}
	return nil
}
