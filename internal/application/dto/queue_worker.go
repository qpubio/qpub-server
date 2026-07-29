package dto

import (
	"encoding/json"
	"time"

	domainWorker "github.com/qpubio/qpub-server/internal/domain/queue/worker"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// WorkerDTO is the REST representation of a registered external worker.
type WorkerDTO struct {
	ID         id.ULID   `json:"id"`
	Name       string    `json:"name"`
	Queues     []string  `json:"queues"`
	LastSeenAt time.Time `json:"last_seen_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func ToWorkerDTO(w domainWorker.Worker) WorkerDTO {
	return WorkerDTO{
		ID:         w.ID,
		Name:       w.Name,
		Queues:     decodeWorkerQueues(w.Queues),
		LastSeenAt: w.LastSeenAt,
		CreatedAt:  w.CreatedAt,
		UpdatedAt:  w.UpdatedAt,
	}
}

// WorkersResponse wraps a list of workers.
type WorkersResponse struct {
	Workers []WorkerDTO `json:"workers"`
}

func ToWorkersDTO(workers []domainWorker.Worker) []WorkerDTO {
	out := make([]WorkerDTO, len(workers))
	for i, w := range workers {
		out[i] = ToWorkerDTO(w)
	}
	return out
}

func decodeWorkerQueues(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var queues []string
	if err := json.Unmarshal([]byte(raw), &queues); err != nil {
		return []string{}
	}
	return queues
}
