package telemetry

import (
	"github.com/qpubio/qpub-server/internal/domain/messaging/backpressure"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Counter holds in-memory telemetry counters for a project/instance pair.
type Counter struct {
	ProjectID         id.Int
	InstanceID        id.ULID
	InboundCount      int64
	InboundRejected   int64
	OutboundCount     int64
	OutboundDropped   int64
	DroppedByReason   map[backpressure.DropReason]int64
	BandwidthInbound  int64
	BandwidthOutbound int64
}

// Snapshot is a point-in-time view of telemetry and gauge stats for Redis sync.
type Snapshot struct {
	ProjectID         id.Int
	InstanceID        id.ULID
	ConnectionCount   int
	ChannelCount      int
	SubscriberCount   int
	InboundCount      int64
	OutboundCount     int64
	OutboundDropped   int64
	BandwidthInbound  int64
	BandwidthOutbound int64
}

// NewCounter creates zero-initialized counters.
func NewCounter(projectID id.Int, instanceID id.ULID) *Counter {
	return &Counter{
		ProjectID:       projectID,
		InstanceID:      instanceID,
		DroppedByReason: make(map[backpressure.DropReason]int64),
	}
}

// NewSnapshot creates zero-initialized snapshot values.
func NewSnapshot(projectID id.Int, instanceID id.ULID) *Snapshot {
	return &Snapshot{
		ProjectID:  projectID,
		InstanceID: instanceID,
	}
}
