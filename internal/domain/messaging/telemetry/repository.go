package telemetry

import (
	"github.com/qpubio/qpub-server/internal/domain/messaging/backpressure"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Repository persists in-memory telemetry counters.
type Repository interface {
	RecordInbound(projectID id.Int, instanceID id.ULID, byteSize int64) error
	RecordOutboundDelivered(projectID id.Int, instanceID id.ULID, byteSize int64) error
	RecordOutboundDropped(projectID id.Int, instanceID id.ULID, reason backpressure.DropReason) error
	RecordInboundRejected(projectID id.Int, instanceID id.ULID) error

	GetCounter(projectID id.Int, instanceID id.ULID) (*Counter, error)
	GetAllForInstance(instanceID id.ULID) ([]*Counter, error)

	ResetForProject(projectID id.Int) error
	ResetForInstance(instanceID id.ULID) error
}
