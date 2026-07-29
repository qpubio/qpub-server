package telemetry

import (
	"time"

	"github.com/qpubio/qpub-server/internal/domain/messaging/backpressure"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Type identifies a telemetry event emitted by the messaging runtime.
type Type string

const (
	TypeInboundAccepted  Type = "inbound.accepted"
	TypeInboundRejected  Type = "inbound.rejected"
	TypeOutboundQueued   Type = "outbound.queued"
	TypeOutboundDelivered Type = "outbound.delivered"
	TypeOutboundDropped  Type = "outbound.dropped"
	TypeOutboundFailed   Type = "outbound.failed"
)

// Event is a canonical telemetry event for the messaging runtime.
type Event struct {
	id         id.ULID
	eventType  Type
	timestamp  time.Time
	projectID  id.Int
	instanceID id.ULID
	data       any
}

// NewEvent creates a telemetry event with the given payload.
func NewEvent(eventType Type, projectID id.Int, instanceID id.ULID, data any) *Event {
	return &Event{
		id:         id.NewULID(),
		eventType:  eventType,
		timestamp:  clock.Now(),
		projectID:  projectID,
		instanceID: instanceID,
		data:       data,
	}
}

func (e *Event) ID() id.ULID         { return e.id }
func (e *Event) Type() Type          { return e.eventType }
func (e *Event) Timestamp() time.Time { return e.timestamp }
func (e *Event) ProjectID() id.Int   { return e.projectID }
func (e *Event) InstanceID() id.ULID { return e.instanceID }
func (e *Event) Data() any           { return e.data }

// InboundAcceptedData is emitted when a publish is accepted.
type InboundAcceptedData struct {
	EnvelopeID   id.ULID
	ConnectionID id.ULID
	Channel      string
	ByteSize     int64
	Source       string
}

// InboundRejectedData is emitted when a publish is rejected.
type InboundRejectedData struct {
	EnvelopeID id.ULID
	Channel    string
	Reason     string
}

// OutboundQueuedData is emitted when a message is queued for delivery.
type OutboundQueuedData struct {
	EnvelopeID     id.ULID
	SubscriptionID id.ULID
	ConnectionID   id.ULID
	Channel        string
	QueueDepth     int
}

// OutboundDeliveredData is emitted when a message is written to the socket.
type OutboundDeliveredData struct {
	EnvelopeID     id.ULID
	SubscriptionID id.ULID
	ConnectionID   id.ULID
	Channel        string
	ByteSize       int64
}

// OutboundDroppedData is emitted when backpressure drops a message.
type OutboundDroppedData struct {
	EnvelopeID     id.ULID
	SubscriptionID id.ULID
	ConnectionID   id.ULID
	Channel        string
	Reason         backpressure.DropReason
	QueueDepth     int
}

// OutboundFailedData is emitted when delivery fails after leaving the queue.
type OutboundFailedData struct {
	EnvelopeID     id.ULID
	SubscriptionID id.ULID
	ConnectionID   id.ULID
	Channel        string
	Reason         string
}

// IsInbound returns true for inbound telemetry events.
func (t Type) IsInbound() bool {
	return t == TypeInboundAccepted || t == TypeInboundRejected
}

// IsOutbound returns true for outbound telemetry events.
func (t Type) IsOutbound() bool {
	switch t {
	case TypeOutboundQueued, TypeOutboundDelivered, TypeOutboundDropped, TypeOutboundFailed:
		return true
	default:
		return false
	}
}

// CountsTowardInbound reports whether the event increments inbound counters.
func (t Type) CountsTowardInbound() bool {
	return t == TypeInboundAccepted
}

// CountsTowardOutbound reports whether the event increments outbound counters.
func (t Type) CountsTowardOutbound() bool {
	return t == TypeOutboundDelivered
}

// CountsTowardDropped reports whether the event increments drop counters.
func (t Type) CountsTowardDropped() bool {
	return t == TypeOutboundDropped
}
