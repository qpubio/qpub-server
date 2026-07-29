package queue

import (
	"encoding/json"
	"time"

	"github.com/qpubio/qpub-server/internal/domain/queue/execution"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

const (
	DefaultVisibilityTimeout = 30 * time.Second
	DefaultMaxAttempts       = 25
	DefaultRetention         = 7 * 24 * time.Hour
)

// Queue represents queue configuration for a project.
type Queue struct {
	ID                 id.Int              `gorm:"primarykey;autoincrement"`
	ProjectID          id.Int              `gorm:"not null;index:idx_queue_project_name,unique"`
	Name               string              `gorm:"not null;index:idx_queue_project_name,unique"`
	ExecutionProfile   execution.Profile   `gorm:"type:string;not null;default:external"`
	VisibilityTimeout  time.Duration       `gorm:"not null;default:30000000000"` // 30s in nanoseconds stored as bigint
	MaxAttempts        int                 `gorm:"not null;default:25"`
	Retention          time.Duration       `gorm:"not null;default:604800000000000"` // 7d
	MaxPayloadBytes    int64               `gorm:"not null;default:1048576"`
	WebhookURL         string              `gorm:"type:text"`
	WebhookSecret      string              `gorm:"type:text"`
	Metadata           json.RawMessage     `gorm:"type:jsonb"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (Queue) TableName() string {
	return "queues"
}

// CreateParams holds parameters for creating a queue.
type CreateParams struct {
	ProjectID         id.Int
	Name              string
	ExecutionProfile  execution.Profile
	VisibilityTimeout time.Duration
	MaxAttempts       int
	Retention         time.Duration
	MaxPayloadBytes   int64
	WebhookURL        string
	WebhookSecret     string
	Metadata          json.RawMessage
}

// UpdateParams holds parameters for updating a queue.
type UpdateParams struct {
	ExecutionProfile  *execution.Profile
	VisibilityTimeout *time.Duration
	MaxAttempts       *int
	Retention         *time.Duration
	MaxPayloadBytes   *int64
	WebhookURL        *string
	WebhookSecret     *string
	Metadata          json.RawMessage
}

// Create creates a validated queue entity.
func Create(params CreateParams) (*Queue, error) {
	if err := ValidateCreate(params); err != nil {
		return nil, err
	}

	visibility := params.VisibilityTimeout
	if visibility <= 0 {
		visibility = DefaultVisibilityTimeout
	}

	maxAttempts := params.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = DefaultMaxAttempts
	}

	retention := params.Retention
	if retention <= 0 {
		retention = DefaultRetention
	}

	maxPayload := params.MaxPayloadBytes
	if maxPayload <= 0 {
		maxPayload = 1024 * 1024
	}

	profile := params.ExecutionProfile
	if profile == "" {
		profile = execution.ProfileExternal
	}

	now := clock.Now()
	return &Queue{
		ProjectID:         params.ProjectID,
		Name:              params.Name,
		ExecutionProfile:  profile,
		VisibilityTimeout: visibility,
		MaxAttempts:       maxAttempts,
		Retention:         retention,
		MaxPayloadBytes:   maxPayload,
		WebhookURL:        params.WebhookURL,
		WebhookSecret:     params.WebhookSecret,
		Metadata:          params.Metadata,
		CreatedAt:           now,
		UpdatedAt:           now,
	}, nil
}

// Update applies changes to the queue entity.
func (q *Queue) Update(params UpdateParams) error {
	if params.ExecutionProfile != nil {
		q.ExecutionProfile = *params.ExecutionProfile
	}
	if params.VisibilityTimeout != nil {
		q.VisibilityTimeout = *params.VisibilityTimeout
	}
	if params.MaxAttempts != nil {
		q.MaxAttempts = *params.MaxAttempts
	}
	if params.Retention != nil {
		q.Retention = *params.Retention
	}
	if params.MaxPayloadBytes != nil {
		q.MaxPayloadBytes = *params.MaxPayloadBytes
	}
	if params.WebhookURL != nil {
		q.WebhookURL = *params.WebhookURL
	}
	if params.WebhookSecret != nil {
		q.WebhookSecret = *params.WebhookSecret
	}
	if params.Metadata != nil {
		q.Metadata = params.Metadata
	}
	q.UpdatedAt = clock.Now()
	return nil
}

// QueueName returns the value object for this queue.
func (q *Queue) QueueName() Name {
	return NewName(q.Name, q.ProjectID)
}
