package dto

import (
	"encoding/json"
	"time"

	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// JobDTO is the REST representation of a job.
type JobDTO struct {
	ID             id.ULID          `json:"id"`
	QueueName      string           `json:"queue_name"`
	Status         domainJob.Status `json:"status"`
	Payload        json.RawMessage  `json:"payload,omitempty"`
	Result         json.RawMessage  `json:"result,omitempty"`
	IdempotencyKey *string          `json:"idempotency_key,omitempty"`
	Attempt        int              `json:"attempt"`
	MaxAttempts    int              `json:"max_attempts"`
	ScheduleAt     *time.Time       `json:"schedule_at,omitempty"`
	StartedAt      *time.Time       `json:"started_at,omitempty"`
	CompletedAt    *time.Time       `json:"completed_at,omitempty"`
	WorkerID       string           `json:"worker_id,omitempty"`
	ErrorMessage   string           `json:"error_message,omitempty"`
	Metadata       json.RawMessage  `json:"metadata,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// EnqueueJobResponse is the JSON body for POST /v1/queue/:queueName/jobs on success.
type EnqueueJobResponse struct {
	JobID  id.ULID          `json:"job_id"`
	Status domainJob.Status `json:"status"`
}

// JobsResponse wraps a list of jobs.
type JobsResponse struct {
	Jobs []JobDTO `json:"jobs"`
}

// JobCountsDTO holds per-status job counts for a queue.
type JobCountsDTO struct {
	Pending   int64 `json:"pending"`
	Scheduled int64 `json:"scheduled"`
	Running   int64 `json:"running"`
	Completed int64 `json:"completed"`
	Failed    int64 `json:"failed"`
	Cancelled int64 `json:"cancelled"`
	DLQ       int64 `json:"dlq"`
}

func ToJobCountsDTO(counts map[domainJob.Status]int64) JobCountsDTO {
	return JobCountsDTO{
		Pending:   counts[domainJob.StatusPending],
		Scheduled: counts[domainJob.StatusScheduled],
		Running:   counts[domainJob.StatusRunning],
		Completed: counts[domainJob.StatusCompleted],
		Failed:    counts[domainJob.StatusFailed],
		Cancelled: counts[domainJob.StatusCancelled],
		DLQ:       counts[domainJob.StatusDLQ],
	}
}

func ToJobDTO(j domainJob.Job) JobDTO {
	return JobDTO{
		ID:             j.ID,
		QueueName:      j.QueueName,
		Status:         j.Status,
		Payload:        j.Payload,
		Result:         j.Result,
		IdempotencyKey: j.IdempotencyKey,
		Attempt:        j.Attempt,
		MaxAttempts:    j.MaxAttempts,
		ScheduleAt:     j.ScheduleAt,
		StartedAt:      j.StartedAt,
		CompletedAt:    j.CompletedAt,
		WorkerID:       j.WorkerID,
		ErrorMessage:   j.ErrorMessage,
		Metadata:       j.Metadata,
		CreatedAt:      j.CreatedAt,
		UpdatedAt:      j.UpdatedAt,
	}
}

func ToJobsDTO(jobs []domainJob.Job) []JobDTO {
	out := make([]JobDTO, len(jobs))
	for i, j := range jobs {
		out[i] = ToJobDTO(j)
	}
	return out
}

func NewEnqueueJobResponse(j *domainJob.Job) EnqueueJobResponse {
	return EnqueueJobResponse{
		JobID:  j.ID,
		Status: j.Status,
	}
}
