package job

import (
	"encoding/json"
	"time"

	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Status represents the lifecycle state of a job.
type Status string

const (
	StatusPending   Status = "pending"
	StatusScheduled Status = "scheduled"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
	StatusDLQ       Status = "dlq"
)

// Job represents a queued work item.
type Job struct {
	ID             id.ULID         `gorm:"type:char(26);primarykey"`
	ProjectID      id.Int          `gorm:"not null;index:idx_job_project_queue_status"`
	QueueName      string          `gorm:"not null;index:idx_job_project_queue_status"`
	Status         Status          `gorm:"not null;index:idx_job_project_queue_status;index:idx_job_schedule"`
	Payload        json.RawMessage `gorm:"type:jsonb"`
	Result         json.RawMessage `gorm:"type:jsonb"`
	IdempotencyKey *string         `gorm:"type:text"`
	Attempt        int             `gorm:"not null;default:0"`
	MaxAttempts    int             `gorm:"not null;default:25"`
	ScheduleAt     *time.Time      `gorm:"index:idx_job_schedule"`
	StartedAt      *time.Time
	CompletedAt    *time.Time
	WorkerID       string
	BrokerSequence uint64
	ErrorMessage   string          `gorm:"type:text"`
	Metadata       json.RawMessage `gorm:"type:jsonb"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (Job) TableName() string {
	return "jobs"
}

// CreateParams holds parameters for creating a job.
type CreateParams struct {
	ProjectID      id.Int
	QueueName      string
	Payload        json.RawMessage
	IdempotencyKey string
	MaxAttempts    int
	ScheduleAt     *time.Time
	Metadata       json.RawMessage
}

// Enqueue creates a new pending or scheduled job.
func Enqueue(params CreateParams) (*Job, error) {
	if err := ValidateCreate(params); err != nil {
		return nil, err
	}

	now := clock.Now()
	status := StatusPending
	if params.ScheduleAt != nil && params.ScheduleAt.After(now) {
		status = StatusScheduled
	}

	maxAttempts := params.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 25
	}

	var idempotencyKey *string
	if params.IdempotencyKey != "" {
		key := params.IdempotencyKey
		idempotencyKey = &key
	}

	return &Job{
		ID:             id.NewULID(),
		ProjectID:      params.ProjectID,
		QueueName:      params.QueueName,
		Status:         status,
		Payload:        params.Payload,
		IdempotencyKey: idempotencyKey,
		MaxAttempts:    maxAttempts,
		ScheduleAt:     params.ScheduleAt,
		Metadata:       params.Metadata,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// MarkRunning transitions a job to running.
func (j *Job) MarkRunning(workerID string) {
	now := clock.Now()
	j.Status = StatusRunning
	j.WorkerID = workerID
	j.Attempt++
	j.StartedAt = &now
	j.UpdatedAt = now
}

// MarkCompleted transitions a job to completed.
func (j *Job) MarkCompleted(result json.RawMessage) {
	now := clock.Now()
	j.Status = StatusCompleted
	j.Result = result
	j.CompletedAt = &now
	j.UpdatedAt = now
}

// MarkFailed transitions a job to failed.
func (j *Job) MarkFailed(reason string) {
	now := clock.Now()
	j.Status = StatusFailed
	j.ErrorMessage = reason
	j.UpdatedAt = now
}

// MarkRetry transitions a job back to pending for retry.
func (j *Job) MarkRetry(delay time.Duration) {
	now := clock.Now()
	scheduleAt := now.Add(delay)
	j.Status = StatusPending
	j.ScheduleAt = &scheduleAt
	j.StartedAt = nil
	j.WorkerID = ""
	j.UpdatedAt = now
}

// MarkDLQ moves a job to the dead-letter state.
func (j *Job) MarkDLQ(reason string) {
	now := clock.Now()
	j.Status = StatusDLQ
	j.ErrorMessage = reason
	j.UpdatedAt = now
}

// MarkCancelled cancels a job.
func (j *Job) MarkCancelled() {
	now := clock.Now()
	j.Status = StatusCancelled
	j.UpdatedAt = now
}

// MarkReclaimed returns a running job to pending after a visibility timeout.
func (j *Job) MarkReclaimed() {
	now := clock.Now()
	j.Status = StatusPending
	j.WorkerID = ""
	j.StartedAt = nil
	j.UpdatedAt = now
}

// CanRetry reports whether the job has remaining attempts.
func (j *Job) CanRetry() bool {
	return j.Attempt < j.MaxAttempts
}

// IsClaimable reports whether the job can be claimed by a worker.
func (j *Job) IsClaimable(now time.Time) bool {
	switch j.Status {
	case StatusPending:
		return j.ScheduleAt == nil || !j.ScheduleAt.After(now)
	case StatusScheduled:
		return j.ScheduleAt != nil && !j.ScheduleAt.After(now)
	default:
		return false
	}
}

// MarkPendingForSchedule promotes a due scheduled job to pending.
func (j *Job) MarkPendingForSchedule() {
	now := clock.Now()
	j.Status = StatusPending
	j.UpdatedAt = now
}
