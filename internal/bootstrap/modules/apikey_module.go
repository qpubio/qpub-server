package modules

import (
	"fmt"
	"reflect"

	apikeyApp "github.com/qpubio/qpub-server/internal/application/service/apikey"
	"github.com/qpubio/qpub-server/internal/bootstrap/container"
	"github.com/qpubio/qpub-server/internal/domain/apikey"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	apikeyRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/apikey"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"gorm.io/gorm"
)

type APIKeyModule struct {
	container.ModuleBase
}

func NewAPIKeyModule() *APIKeyModule {
	return &APIKeyModule{
		ModuleBase: container.NewModule("apikey", 31),
	}
}

func (m *APIKeyModule) Register(c *container.Container) error {
	c.Register(
		reflect.TypeOf((*apikey.Repository)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			db, err := container.GetTyped[*gorm.DB](c)
			if err != nil {
				return nil, err
			}
			return apikeyRepo.NewRepository(db), nil
		},
	)

	c.Register(
		reflect.TypeOf((*apikey.Service)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}
			repository, err := container.GetTyped[apikey.Repository](c)
			if err != nil {
				return nil, err
			}
			return apikeyApp.NewService(logger, repository), nil
		},
	)

	c.Register(
		reflect.TypeOf((*apikeyApp.ExtendedService)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			service, err := container.GetTyped[apikey.Service](c)
			if err != nil {
				return nil, err
			}
			extended, ok := service.(apikeyApp.ExtendedService)
			if !ok {
				return nil, fmt.Errorf("apikey service does not satisfy ExtendedService")
			}
			return extended, nil
		},
	)

	if logger, err := container.GetTyped[logger.Service](c); err == nil {
		logger.Info(log.App, "API key module registered successfully")
	}
	return nil
}
