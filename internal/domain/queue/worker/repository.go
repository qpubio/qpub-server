package worker

import "github.com/qpubio/qpub-server/internal/shared/id"

type Repository interface {
	Create(worker *Worker) error
	Update(worker *Worker) error
	FindByID(projectID id.Int, workerID id.ULID) (*Worker, error)
	ListByProject(projectID id.Int) ([]Worker, error)
}

type Service interface {
	Register(params CreateParams) (Worker, error)
	Heartbeat(projectID id.Int, workerID id.ULID) (Worker, error)
	Get(projectID id.Int, workerID id.ULID) (Worker, error)
	ListByProject(projectID id.Int) ([]Worker, error)
}
