package job

import (
	"encoding/json"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"time"
)

// Service defines job management use cases.
type Service interface {
	Get(projectID id.Int, queueName string, jobID id.ULID) (Job, error)
	List(filter ListFilter) ([]Job, error)
	CountByStatus(projectID id.Int, queueName string) (map[Status]int64, error)
	Cancel(projectID id.Int, queueName string, jobID id.ULID) (Job, error)
	Retry(projectID id.Int, queueName string, jobID id.ULID) (Job, error)
	UpdateProgress(projectID id.Int, queueName string, jobID id.ULID, metadata json.RawMessage) (Job, error)
}

// EnqueueRequest holds parameters for enqueueing a job through the router.
type EnqueueRequest struct {
	ProjectID      id.Int
	QueueName      string
	Payload        json.RawMessage
	IdempotencyKey string
	ScheduleAt     *time.Time
	Delay          time.Duration
	Metadata       json.RawMessage
}

// DequeueRequest holds parameters for pulling jobs.
type DequeueRequest struct {
	ProjectID id.Int
	QueueName string
	WorkerID  string
	BatchSize int
	Wait      time.Duration
}

// AckRequest holds parameters for acknowledging a job.
type AckRequest struct {
	ProjectID id.Int
	QueueName string
	JobID     id.ULID
	WorkerID  string
	Result    json.RawMessage
}

// NackRequest holds parameters for negative acknowledgment.
type NackRequest struct {
	ProjectID id.Int
	QueueName string
	JobID     id.ULID
	WorkerID  string
	Reason    string
	RetryDelay time.Duration
}
