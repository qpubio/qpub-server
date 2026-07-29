package connection

import (
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// MessageHandler is a function that processes incoming messages
type MessageHandler func(connID id.ULID, message []byte)

// Service defines the connection service interface
type Service interface {
	// Register adds a new connection with a send handler
	Register(conn *Connection, sendHandler func([]byte) error) error

	// Unregister removes a WebSocket connection
	Unregister(connID id.ULID) error

	// Send sends a message to a connection
	Send(connID id.ULID, message []byte) error

	// Broadcast sends a message to all connections for a project
	Broadcast(projID id.Int, message []byte) error

	// Close closes a connection
	Close(connID id.ULID) error

	// CloseAllByProject closes all connections for a project
	CloseAllByProject(projID id.Int) error

	// CleanStaleConnections removes connections that haven't received a pong in the specified duration
	CleanStaleConnections() (int, error)

	// Get returns a connection by ID
	Get(connID id.ULID) (*Connection, error)
}
