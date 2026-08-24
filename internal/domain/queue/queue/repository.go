package queue

import (
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Repository defines persistence operations for queues.
type Repository interface {
	Create(queue *Queue) (id.Int, error)
	Update(queue *Queue) error
	FindByProjectAndName(projectID id.Int, name string) (*Queue, error)
	FindByID(id id.Int) (*Queue, error)
	ListByProjectPaginated(projectID id.Int, limit, offset int) ([]Queue, int64, error)
}
