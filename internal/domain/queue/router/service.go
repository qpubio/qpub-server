package router

import (
	"context"

	"github.com/qpubio/qpub-server/internal/domain/queue/job"
	"github.com/qpubio/qpub-server/internal/domain/queue/receipt"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Service is the single hot path for queue operations.
type Service interface {
	Enqueue(ctx context.Context, req job.EnqueueRequest) (*receipt.Receipt, *job.Job, error)
	Dequeue(ctx context.Context, req job.DequeueRequest) ([]job.Job, error)
	Ack(ctx context.Context, req job.AckRequest) (*receipt.Receipt, error)
	Nack(ctx context.Context, req job.NackRequest) (*receipt.Receipt, error)
	Cancel(ctx context.Context, projectID id.Int, queueName string, jobID id.ULID) (*receipt.Receipt, *job.Job, error)
	EnsureQueueListening(projectID id.Int, queueName string) error
	PublishDueScheduled(ctx context.Context) error
	ReclaimExpired(ctx context.Context, limit int) (int64, error)
}
