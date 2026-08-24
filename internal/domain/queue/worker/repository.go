package worker

import "github.com/qpubio/qpub-server/internal/shared/id"

type Repository interface {
	Create(worker *Worker) error
	Update(worker *Worker) error
	FindByID(projectID id.Int, workerID id.ULID) (*Worker, error)
	ListByProjectPaginated(projectID id.Int, limit, offset int) ([]Worker, int64, error)
}

type Service interface {
	Register(params CreateParams) (Worker, error)
	Heartbeat(projectID id.Int, workerID id.ULID) (Worker, error)
	Get(projectID id.Int, workerID id.ULID) (Worker, error)
	ListByProjectPaginated(projectID id.Int, page, perPage int) ([]Worker, int64, error)
}
