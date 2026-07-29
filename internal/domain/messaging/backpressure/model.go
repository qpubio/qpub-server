package backpressure

// Point identifies where backpressure is applied in the runtime.
type Point string

const (
	PointSubscriptionBuffer  Point = "subscription_buffer"
	PointConnectionOutbound  Point = "connection_outbound"
	PointProjectInboundRate  Point = "project_inbound_rate"
	PointProjectOutboundRate Point = "project_outbound_rate"
)

// DropReason explains why a message was dropped.
type DropReason string

const (
	DropReasonBufferFull        DropReason = "buffer_full"
	DropReasonWriteTimeout      DropReason = "write_timeout"
	DropReasonSlowConsumer      DropReason = "slow_consumer"
	DropReasonInboundRateLimit  DropReason = "inbound_rate_limit"
	DropReasonOutboundRateLimit DropReason = "outbound_rate_limit"
)

// String returns the string representation of the drop reason.
func (r DropReason) String() string {
	return string(r)
}

// Result is the outcome of a backpressure-aware enqueue attempt.
type Result struct {
	Queued     bool
	Dropped    bool
	Reason     DropReason
	QueueDepth int
}

// Queue is a bounded outbound queue abstraction used by policies.
type Queue interface {
	Depth() int
	Capacity() int
	Push(item []byte) bool
}

// BoundedQueue is an in-memory fixed-capacity queue for outbound payloads.
type BoundedQueue struct {
	items    [][]byte
	capacity int
}

// NewBoundedQueue creates a queue with the given capacity.
func NewBoundedQueue(capacity int) *BoundedQueue {
	if capacity < 1 {
		capacity = 1
	}
	return &BoundedQueue{
		items:    make([][]byte, 0, capacity),
		capacity: capacity,
	}
}

func (q *BoundedQueue) Depth() int {
	if q == nil {
		return 0
	}
	return len(q.items)
}

func (q *BoundedQueue) Capacity() int {
	if q == nil {
		return 0
	}
	return q.capacity
}

// Push appends an item when capacity allows.
func (q *BoundedQueue) Push(item []byte) bool {
	if q == nil || len(q.items) >= q.capacity {
		return false
	}
	q.items = append(q.items, append([]byte(nil), item...))
	return true
}

// Pop removes and returns the oldest item.
func (q *BoundedQueue) Pop() ([]byte, bool) {
	if q == nil || len(q.items) == 0 {
		return nil, false
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

// Items returns a snapshot of queued payloads.
func (q *BoundedQueue) Items() [][]byte {
	if q == nil {
		return nil
	}
	out := make([][]byte, len(q.items))
	for i, item := range q.items {
		out[i] = append([]byte(nil), item...)
	}
	return out
}
