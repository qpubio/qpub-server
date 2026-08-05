package modules

import (
	"reflect"

	tenantApp "github.com/qpubio/qpub-server/internal/application/service/tenant"
	"github.com/qpubio/qpub-server/internal/bootstrap/container"
	"github.com/qpubio/qpub-server/internal/domain/tenant"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	tenantRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/tenant"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"gorm.io/gorm"
)

type TenantModule struct {
	container.ModuleBase
}

func NewTenantModule() *TenantModule {
	return &TenantModule{
		ModuleBase: container.NewModule("tenant", 39), // Before messaging (40)
	}
}

func (m *TenantModule) Register(c *container.Container) error {
	c.Register(
		reflect.TypeOf((*tenant.Repository)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			db, err := container.GetTyped[*gorm.DB](c)
			if err != nil {
				return nil, err
			}
			return tenantRepo.NewRepository(db), nil
		},
	)

	c.Register(reflect.TypeOf((*tenant.Service)(nil)).Elem(), func(c *container.Container) (interface{}, error) {
		logger, err := container.GetTyped[logger.Service](c)
		if err != nil {
			return nil, err
		}
		repo, err := container.GetTyped[tenant.Repository](c)
		if err != nil {
			return nil, err
		}
		return tenantApp.NewService(logger, repo), nil
	})

	if logger, err := container.GetTyped[logger.Service](c); err == nil {
		logger.Info(log.App, "Tenant module registered successfully")
	}
	return nil
}
