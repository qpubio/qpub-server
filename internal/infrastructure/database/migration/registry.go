package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
)

var migrations []*gormigrate.Migration

// registerMigration adds a migration to the registry
func registerMigration(m *gormigrate.Migration) {
	migrations = append(migrations, m)
}

// GetMigrations returns all registered migrations
func GetMigrations() []*gormigrate.Migration {
	return migrations
}
