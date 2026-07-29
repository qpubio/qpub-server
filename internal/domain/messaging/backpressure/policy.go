package backpressure

import "github.com/qpubio/qpub-server/internal/domain/messaging/envelope"

// Policy decides whether an envelope may enter a queue under pressure.
type Policy interface {
	TryEnqueue(queue Queue, env *envelope.Envelope) Result
}

// DropNewestPolicy drops the incoming message when the queue is full.
type DropNewestPolicy struct {
	Point Point
}

// NewDropNewestPolicy creates a drop-newest policy for a pressure point.
func NewDropNewestPolicy(point Point) *DropNewestPolicy {
	return &DropNewestPolicy{Point: point}
}

// TryEnqueue enqueues the payload or drops the newest message when full.
func (p *DropNewestPolicy) TryEnqueue(queue Queue, env *envelope.Envelope) Result {
	if queue == nil || env == nil {
		return Result{Dropped: true, Reason: DropReasonBufferFull}
	}

	payload := env.Payload()
	if queue.Push(payload) {
		return Result{
			Queued:     true,
			QueueDepth: queue.Depth(),
		}
	}

	return Result{
		Dropped:    true,
		Reason:     DropReasonBufferFull,
		QueueDepth: queue.Depth(),
	}
}

// BlockWithTimeoutPolicy is a placeholder policy type for connection outbound.
// Phase 5 will add timed blocking; Phase 1 defines the shape only.
type BlockWithTimeoutPolicy struct {
	Point          Point
	TimeoutMillis  int
	FallbackReason DropReason
}

// NewBlockWithTimeoutPolicy creates a block-with-timeout policy definition.
func NewBlockWithTimeoutPolicy(point Point, timeoutMillis int) *BlockWithTimeoutPolicy {
	return &BlockWithTimeoutPolicy{
		Point:          point,
		TimeoutMillis:  timeoutMillis,
		FallbackReason: DropReasonWriteTimeout,
	}
}

// TryEnqueue currently behaves as drop-newest until Phase 5 adds timed blocking.
func (p *BlockWithTimeoutPolicy) TryEnqueue(queue Queue, env *envelope.Envelope) Result {
	dropNewest := NewDropNewestPolicy(p.Point)
	result := dropNewest.TryEnqueue(queue, env)
	if result.Dropped {
		result.Reason = p.FallbackReason
	}
	return result
}
