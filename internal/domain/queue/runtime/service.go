package runtime

import "context"

// JobHandler processes a job payload for platform or registered handlers.
type JobHandler func(ctx context.Context, payload []byte) error

// Service manages the queue runtime lifecycle.
type Service interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	RegisterHandler(queueName string, handler JobHandler)
}
