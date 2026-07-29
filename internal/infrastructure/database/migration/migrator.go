package migration

import (
	"fmt"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"sort"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

type Migrator interface {
	Migrate() error
}

type migrator struct {
	db     *gorm.DB
	logger logger.Service
}

func New(db *gorm.DB, logger logger.Service) Migrator {
	return &migrator{
		db:     db,
		logger: logger,
	}
}

func (m *migrator) Migrate() error {
	// Get all migrations in order
	migrations := GetMigrations()

	// Sort migrations by ID to ensure correct order
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})

	// Initialize gormigrate with migrations
	gormigrateInstance := gormigrate.New(m.db, gormigrate.DefaultOptions, migrations)

	// Run migrations
	if err := gormigrateInstance.Migrate(); err != nil {
		m.logger.Error(log.Migration, "Failed to run migrations: %v", err)
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	m.logger.Info(log.Migration, "Successfully ran all migrations")
	return nil
}
