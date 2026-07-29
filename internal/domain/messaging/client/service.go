package client

import (
	"encoding/json"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Service defines the client service interface
type Service interface {
	// Connect registers a new client connection
	Connect(
		connID id.ULID,
		projID id.Int,
		apiKeyID id.Int,
		clientID *string,
		permission *json.RawMessage,
	) (*Client, error)

	// Disconnect marks a client as disconnected
	Disconnect(connID id.ULID) error

	// GetClient retrieves a client by connection ID
	GetClient(connID id.ULID) (*Client, error)

	// SendMessage sends a message to a client
	SendMessage(connID id.ULID, message []byte) error

	// BroadcastToProject sends a message to all clients in a project
	BroadcastToProject(projID id.Int, message []byte) error

	// CleanDisconnectedClients removes disconnected clients
	CleanDisconnectedClients() (int, error)
}
