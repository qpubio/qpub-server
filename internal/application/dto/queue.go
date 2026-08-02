package dto

import (
	"encoding/json"
	"time"

	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
)

// QueueDTO is the REST/control representation of a queue config (snake_case JSON).
type QueueDTO struct {
	Name              string          `json:"name"`
	ExecutionProfile  string          `json:"execution_profile"`
	VisibilityTimeout string          `json:"visibility_timeout"`
	MaxAttempts       int             `json:"max_attempts"`
	Retention         string          `json:"retention"`
	MaxPayloadBytes   int64           `json:"max_payload_bytes"`
	WebhookURL        string          `json:"webhook_url"`
	WebhookSecret     string          `json:"webhook_secret,omitempty"`
	Metadata          json.RawMessage `json:"metadata,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func ToQueueDTO(q domainQueue.Queue) QueueDTO {
	return QueueDTO{
		Name:              q.Name,
		ExecutionProfile:  string(q.ExecutionProfile),
		VisibilityTimeout: q.VisibilityTimeout.String(),
		MaxAttempts:       q.MaxAttempts,
		Retention:         q.Retention.String(),
		MaxPayloadBytes:   q.MaxPayloadBytes,
		WebhookURL:        q.WebhookURL,
		WebhookSecret:     q.WebhookSecret,
		Metadata:          q.Metadata,
		CreatedAt:         q.CreatedAt,
		UpdatedAt:         q.UpdatedAt,
	}
}

// QueueSummaryDTO is a queue config with per-status job counts (control API).
type QueueSummaryDTO struct {
	QueueDTO
	Counts JobCountsDTO `json:"counts"`
}

// QueuesResponse wraps a list of queues.
type QueuesResponse struct {
	Queues []QueueSummaryDTO `json:"queues"`
}

func ToQueueSummaryDTO(q domainQueue.Queue, counts map[domainJob.Status]int64) QueueSummaryDTO {
	return QueueSummaryDTO{
		QueueDTO: ToQueueDTO(q),
		Counts:   ToJobCountsDTO(counts),
	}
}
