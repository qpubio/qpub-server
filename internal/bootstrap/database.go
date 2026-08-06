package bootstrap

import (
	"fmt"
	"time"

	"github.com/qpubio/qpub-server/internal/infrastructure/database"
	"github.com/qpubio/qpub-server/internal/infrastructure/database/migration"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"github.com/go-redis/redis"
)

const (
	// Namespaced per service: qpub-backend uses the same Redis and must not share this key.
	dbMigrateLockKey     = "lock:db:migrate:server"
	dbMigrateLockTimeout = 5 * time.Minute
)

func setupDatabase(app *App) error {
	dbService, err := database.New(&app.config.Infrastructure.Database, app.logger)
	if err != nil {
		return fmt.Errorf("failed to create database service: %w", err)
	}
	if _, err := dbService.Connect(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	app.cleanup.Register(func() error {
		if err := dbService.Close(); err != nil {
			app.logger.Error(log.DB, "Error closing database connection: %v", err)
			return err
		}
		return nil
	})
	app.db = dbService.DB()
	if err := runMigrations(app); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

func runMigrations(app *App) error {
	locked, err := app.redis.SetNX(dbMigrateLockKey, app.instanceID, dbMigrateLockTimeout).Result()
	if err != nil {
		return fmt.Errorf("failed to acquire migrate lock: %w", err)
	}
	if !locked {
		return waitForMigrations(app)
	}
	defer app.redis.Del(dbMigrateLockKey)

	migrator := migration.New(app.db, app.logger)
	if err := migrator.Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

func waitForMigrations(app *App) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timeout := time.After(dbMigrateLockTimeout)

	for {
		select {
		case <-app.ctx.Done():
			return app.ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for migrations")
		case <-ticker.C:
			_, err := app.redis.Get(dbMigrateLockKey).Result()
			if err == redis.Nil {
				return nil
			}
			if err != nil {
				app.logger.Error(log.DB, "Failed to check migrate lock: %v", err)
				continue
			}
		}
	}
}
