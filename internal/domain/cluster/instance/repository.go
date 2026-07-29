package instance

import (
	"github.com/qpubio/qpub-server/internal/shared/id"
	"time"
)

// Repository defines the interface for server instance persistence operations
type Repository interface {
	// Create creates a new server instance
	Create(instance *ServerInstance) (id.Int, error)

	// Update updates an existing server instance
	Update(instance *ServerInstance) error

	// Delete removes a server instance
	Delete(instance *ServerInstance) error

	// FindByID finds a server instance by ID
	FindByID(id id.Int) (*ServerInstance, error)

	// FindByInstanceID finds a server instance by instance ID
	FindByInstanceID(instanceID id.ULID) (*ServerInstance, error)

	// ListActive returns all active server instances
	ListActive() ([]*ServerInstance, error)

	// ListStale returns all active server instances with last heartbeat
	// older than the given threshold
	ListStale(threshold time.Time) ([]*ServerInstance, error)
}
