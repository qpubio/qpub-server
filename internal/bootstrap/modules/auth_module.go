package modules

import (
	"fmt"
	"reflect"

	authAPIKeyService "github.com/qpubio/qpub-server/internal/application/service/auth/apikey"
	authTokenService "github.com/qpubio/qpub-server/internal/application/service/auth/token"
	"github.com/qpubio/qpub-server/internal/bootstrap/container"
	"github.com/qpubio/qpub-server/internal/config"
	"github.com/qpubio/qpub-server/internal/domain/apikey"
	authAPIKey "github.com/qpubio/qpub-server/internal/domain/auth/apikey"
	authToken "github.com/qpubio/qpub-server/internal/domain/auth/token"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	tokenRepo "github.com/qpubio/qpub-server/internal/infrastructure/repository/auth/token"
	sharedAPIKey "github.com/qpubio/qpub-server/internal/shared/apikey"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"gorm.io/gorm"
)

type AuthModule struct {
	container.ModuleBase
}

func NewAuthModule() *AuthModule {
	return &AuthModule{
		ModuleBase: container.NewModule("auth", 60),
	}
}

func (m *AuthModule) Register(c *container.Container) error {
	c.Register(
		reflect.TypeOf((*authToken.Repository)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			db, err := container.GetTyped[*gorm.DB](c)
			if err != nil {
				return nil, err
			}
			return tokenRepo.NewRepository(db), nil
		},
	)

	c.Register(
		reflect.TypeOf((*authToken.Service)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			cfg, err := container.GetTyped[*config.Config](c)
			if err != nil {
				return nil, err
			}
			repository, err := container.GetTyped[authToken.Repository](c)
			if err != nil {
				return nil, err
			}
			apiKeyService, err := container.GetTyped[apikey.Service](c)
			if err != nil {
				return nil, fmt.Errorf("apikey service is required for token service: %w", err)
			}
			apikeyParser, err := container.GetTyped[*sharedAPIKey.Parser](c)
			if err != nil {
				return nil, fmt.Errorf("API key parser is required for token service: %w", err)
			}
			return authTokenService.NewService(cfg, repository, apiKeyService, apikeyParser), nil
		},
	)

	c.Register(
		reflect.TypeOf((*authAPIKey.Service)(nil)).Elem(),
		func(c *container.Container) (interface{}, error) {
			logger, err := container.GetTyped[logger.Service](c)
			if err != nil {
				return nil, err
			}
			apiKeyService, err := container.GetTyped[apikey.Service](c)
			if err != nil {
				return nil, err
			}
			apikeyParser, err := container.GetTyped[*sharedAPIKey.Parser](c)
			if err != nil {
				return nil, err
			}
			return authAPIKeyService.NewService(logger, apiKeyService, apikeyParser), nil
		},
	)

	if logger, err := container.GetTyped[logger.Service](c); err == nil {
		logger.Info(log.App, "Auth module registered successfully")
	}
	return nil
}
