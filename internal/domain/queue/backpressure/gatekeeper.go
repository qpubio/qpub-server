package backpressure

import "github.com/qpubio/qpub-server/internal/shared/id"

type Gatekeeper interface {
	AllowEnqueue(projectID id.Int) (bool, error)
	AllowDequeue(projectID id.Int) (bool, error)
}
