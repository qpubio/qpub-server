package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func init() {
	registerMigration(&gormigrate.Migration{
		ID: "2026091501_jobs_list_index",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`
				CREATE INDEX IF NOT EXISTS idx_job_project_queue_created
				ON jobs (project_id, queue_name, created_at DESC)
			`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`DROP INDEX IF EXISTS idx_job_project_queue_created`).Error
		},
	})
}
