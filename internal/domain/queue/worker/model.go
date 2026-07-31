package worker

import (
	"encoding/json"
	"time"

	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Worker represents a registered external worker.
type Worker struct {
	ID          id.ULID `gorm:"type:char(22);primarykey"`
	ProjectID   id.Int  `gorm:"not null;index"`
	Name        string  `gorm:"not null"`
	Queues      string  `gorm:"type:text"`
	LastSeenAt  time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Worker) TableName() string {
	return "queue_workers"
}

type CreateParams struct {
	ProjectID id.Int
	Name      string
	Queues    []string
}

func Create(params CreateParams) (*Worker, error) {
	now := clock.Now()
	return &Worker{
		ID:         id.NewULID(),
		ProjectID:  params.ProjectID,
		Name:       params.Name,
		Queues:     encodeQueues(params.Queues),
		LastSeenAt: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func encodeQueues(queues []string) string {
	if len(queues) == 0 {
		return ""
	}
	data, err := json.Marshal(queues)
	if err != nil {
		return ""
	}
	return string(data)
}

func (w *Worker) Heartbeat() {
	now := clock.Now()
	w.LastSeenAt = now
	w.UpdatedAt = now
}
