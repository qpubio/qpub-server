package modules

import (
	"fmt"
	"reflect"

	"github.com/qpubio/qpub-server/internal/bootstrap/container"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/infrastructure/nats"
	"github.com/qpubio/qpub-server/internal/infrastructure/redis"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	goredis "github.com/go-redis/redis"
	gonats "github.com/nats-io/nats.go"
)

// InfrastructureModule provides registration for infrastructure services
type InfrastructureModule struct {
	container.ModuleBase
}

// NewInfrastructureModule creates a new infrastructure module
func NewInfrastructureModule() *InfrastructureModule {
	return &InfrastructureModule{
		ModuleBase: container.NewModule("infrastructure", 1), // Lowest order number = initialize first
	}
}

// Register registers all infrastructure services
func (m *InfrastructureModule) Register(c *container.Container) error {
	// Register Redis client from the infrastructure.Redis service
	c.Register(
		reflect.TypeOf((*goredis.Client)(nil)),
		func(c *container.Container) (interface{}, error) {
			// Get the redis service directly from the container
			redisService, err := container.GetTyped[redis.Service](c)
			if err != nil {
				logger, _ := container.GetTyped[logger.Service](c)
				if logger != nil {
					logger.Error(log.App, "Failed to get Redis service: %v", err)
				}
				return nil, err
			}

			// Return the Redis client
			client := redisService.Client()
			if client == nil {
				logger, _ := container.GetTyped[logger.Service](c)
				if logger != nil {
					logger.Error(log.App, "Redis client is nil")
				}
				return nil, fmt.Errorf("redis client is nil")
			}

			return client, nil
		},
	)

	// Register NATS connection from the infrastructure.NATS service
	c.Register(
		reflect.TypeOf((*gonats.Conn)(nil)),
		func(c *container.Container) (interface{}, error) {
			// Get the NATS service directly from the container
			natsService, err := container.GetTyped[nats.Service](c)
			if err != nil {
				logger, _ := container.GetTyped[logger.Service](c)
				if logger != nil {
					logger.Error(log.App, "Failed to get NATS service: %v", err)
				}
				return nil, err
			}

			// Return the NATS connection
			conn := natsService.Conn()
			if conn == nil {
				logger, _ := container.GetTyped[logger.Service](c)
				if logger != nil {
					logger.Error(log.App, "NATS connection is nil")
				}
				return nil, fmt.Errorf("nats connection is nil")
			}

			return conn, nil
		},
	)

	logger, err := container.GetTyped[logger.Service](c)
	if err == nil {
		logger.Info(log.App, "Infrastructure module registered successfully")
	}

	return nil
}
