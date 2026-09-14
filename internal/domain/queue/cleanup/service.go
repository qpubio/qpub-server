package cleanup

import (
	"context"

	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Service orchestrates queue/tenant deletion and retention maintenance.
type Service interface {
	RequestDeleteQueue(ctx context.Context, projectID id.Int, queueName string, force bool) (completedSync bool, err error)
	RequestDeleteTenant(ctx context.Context, tenantID id.Int) error
	PurgeTerminalJobs(ctx context.Context) error
	ResumePendingCascades(ctx context.Context) error
	PurgeStaleWorkers(ctx context.Context) (int, error)
}
