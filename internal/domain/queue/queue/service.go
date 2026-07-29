package queue

import "github.com/qpubio/qpub-server/internal/shared/id"

// Service defines queue management use cases.
type Service interface {
	Create(params CreateParams) (Queue, error)
	Update(projectID id.Int, name string, params UpdateParams) (Queue, error)
	Get(projectID id.Int, name string) (Queue, error)
	List(projectID id.Int) ([]Queue, error)
	Ensure(params CreateParams) (Queue, error)
}
