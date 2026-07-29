package connection

import (
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Repository defines the connection repository interface
type Repository interface {
	// Store adds or updates a connection
	Store(conn *Connection) error

	// Remove removes a connection
	Remove(connID id.ULID) error

	// FindByID finds a connection by ID
	FindByID(connID id.ULID) (*Connection, error)

	// FindAllByProjectID finds all connections for a project
	FindAllByProjectID(projID id.Int) ([]*Connection, error)

	// CountByProject returns the number of connections for a specific project
	CountByProject(projID id.Int) int

	// CleanStaleConnections removes connections that haven't received a pong in the specified duration
	CleanStaleConnections() int
}
