package worker

import (
	"time"

	"github.com/qpubio/qpub-server/internal/shared/id"
)

type Repository interface {
	Create(worker *Worker) error
	Update(worker *Worker) error
	Delete(projectID id.Int, workerID id.ULID) error
	FindByID(projectID id.Int, workerID id.ULID) (*Worker, error)
	ListByProjectPaginated(projectID id.Int, limit, offset int) ([]Worker, int64, error)
	ListByProject(projectID id.Int) ([]Worker, error)
	ListStale(before time.Time, limit int) ([]Worker, error)
	HasRunningJobs(projectID id.Int, workerID string) (bool, error)
}

type Service interface {
	Register(params CreateParams) (Worker, error)
	Heartbeat(projectID id.Int, workerID id.ULID) (Worker, error)
	Get(projectID id.Int, workerID id.ULID) (Worker, error)
	ListByProjectPaginated(projectID id.Int, page, perPage int) ([]Worker, int64, error)
}
