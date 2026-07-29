package telemetry

import "github.com/qpubio/qpub-server/internal/shared/id"

// Service records telemetry events and exposes counter snapshots.
type Service interface {
	Record(evt *Event) error
	GetSnapshots(instanceID id.ULID) ([]*Snapshot, error)
	ResetForProject(projectID id.Int) error
}
