package modules

import (
	"reflect"

	tenantApp "github.com/qpubio/qpub-server/internal/application/service/tenant"
	"github.com/qpubio/qpub-server/internal/bootstrap/container"
	"github.com/qpubio/qpub-server/internal/domain/tenant"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
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
	c.Register(reflect.TypeOf((*tenant.Service)(nil)).Elem(), func(c *container.Container) (interface{}, error) {
		logger, err := container.GetTyped[logger.Service](c)
		if err != nil {
			return nil, err
		}
		// In-memory store; optional DB repo can be added later without changing the port.
		return tenantApp.NewService(logger, nil), nil
	})

	if logger, err := container.GetTyped[logger.Service](c); err == nil {
		logger.Info(log.App, "Tenant module registered successfully")
	}
	return nil
}
