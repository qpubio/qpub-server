package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func init() {
	registerMigration(&gormigrate.Migration{
		ID: "2026080102_queue_cleanup",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Exec(`
				ALTER TABLE tenants ADD COLUMN IF NOT EXISTS status STRING NOT NULL DEFAULT 'active'
			`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`
				ALTER TABLE queues ADD COLUMN IF NOT EXISTS status STRING NOT NULL DEFAULT 'active'
			`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`
				ALTER TABLE jobs ADD COLUMN IF NOT EXISTS terminal_at TIMESTAMPTZ
			`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`
				UPDATE jobs
				SET terminal_at = COALESCE(completed_at, updated_at)
				WHERE status IN ('completed', 'cancelled', 'failed', 'dlq')
				  AND terminal_at IS NULL
			`).Error; err != nil {
				return err
			}
			return tx.Exec(`
				CREATE INDEX IF NOT EXISTS idx_job_status_terminal_at
				ON jobs (status, terminal_at)
				WHERE terminal_at IS NOT NULL
			`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Exec(`DROP INDEX IF EXISTS idx_job_status_terminal_at`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`ALTER TABLE jobs DROP COLUMN IF EXISTS terminal_at`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`ALTER TABLE queues DROP COLUMN IF EXISTS status`).Error; err != nil {
				return err
			}
			return tx.Exec(`ALTER TABLE tenants DROP COLUMN IF EXISTS status`).Error
		},
	})
}
