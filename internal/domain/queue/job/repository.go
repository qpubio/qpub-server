package job

import (
	"github.com/qpubio/qpub-server/internal/shared/id"
	"time"
)

// ListFilter defines query filters for listing jobs.
type ListFilter struct {
	ProjectID id.Int
	QueueName string
	Status    Status
	Limit     int
	Offset    int
}

// Repository defines persistence operations for jobs.
type Repository interface {
	Create(job *Job) error
	Update(job *Job) error
	FindByID(projectID id.Int, queueName string, jobID id.ULID) (*Job, error)
	FindByIdempotencyKey(projectID id.Int, queueName, key string) (*Job, error)
	List(filter ListFilter) ([]Job, error)
	FindDueScheduled(limit int, now time.Time) ([]Job, error)
	CountByStatus(projectID id.Int, queueName string, status Status) (int64, error)
	// ClaimPending atomically claims up to limit pending jobs for the queue.
	ClaimPending(projectID id.Int, queueName, workerID string, limit int, now time.Time) ([]Job, error)
	// ReclaimExpired resets running jobs whose visibility lease has expired.
	ReclaimExpired(now time.Time, limit int, defaultVisibility time.Duration) (int64, error)
	// ExtendLease refreshes started_at for running jobs held by workerID.
	ExtendLease(projectID id.Int, workerID string, now time.Time) (int64, error)
}
