package bootstrap

import (
	"fmt"

	"github.com/qpubio/qpub-server/internal/infrastructure/redis"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

func setupRedis(app *App) error {
	redisService, err := redis.New(&app.config.Infrastructure.Redis)
	if err != nil {
		return fmt.Errorf("failed to create redis service: %w", err)
	}
	if err := redisService.Ping(app.ctx); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}
	app.redis = redisService
	app.cleanup.Register(func() error {
		app.logger.Info(log.Redis, "Closing Redis connection...")
		return redisService.Close()
	})
	app.logger.Info(log.Redis, "Redis initialized successfully at %s:%s",
		app.config.Infrastructure.Redis.Host,
		app.config.Infrastructure.Redis.Port,
	)
	return nil
}
