package migration

import (
	"github.com/qpubio/qpub-server/internal/domain/apikey"
	"github.com/qpubio/qpub-server/internal/domain/auth/token"
	"github.com/qpubio/qpub-server/internal/domain/cluster/instance"
	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
	domainWorker "github.com/qpubio/qpub-server/internal/domain/queue/worker"
	"github.com/qpubio/qpub-server/internal/domain/tenant"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func init() {
	registerMigration(&gormigrate.Migration{
		ID: "2026080101_dataplane_tables",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(
				&tenant.Tenant{},
				&tenant.Limits{},
				&apikey.APIKey{},
				&token.RevokedToken{},
				&domainQueue.Queue{},
				&domainJob.Job{},
				&domainWorker.Worker{},
				&instance.ServerInstance{},
			); err != nil {
				return err
			}

			// Partial unique index: enforce idempotency per project+queue only when a key is set.
			return tx.Exec(`
				CREATE UNIQUE INDEX IF NOT EXISTS idx_job_idempotency
				ON jobs (project_id, queue_name, idempotency_key)
				WHERE idempotency_key IS NOT NULL AND idempotency_key <> ''
			`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Exec(`DROP INDEX IF EXISTS idx_job_idempotency`).Error; err != nil {
				return err
			}
			return tx.Migrator().DropTable(
				instance.ServerInstance{}.TableName(),
				domainWorker.Worker{}.TableName(),
				domainJob.Job{}.TableName(),
				domainQueue.Queue{}.TableName(),
				token.RevokedToken{}.TableName(),
				apikey.APIKey{}.TableName(),
				tenant.Limits{}.TableName(),
				tenant.Tenant{}.TableName(),
			)
		},
	})
}
