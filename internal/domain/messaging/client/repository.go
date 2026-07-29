package client

import (
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Repository defines the client repository interface
type Repository interface {
	// Create adds a new client
	Create(client *Client) error

	// Update updates an existing client
	Update(client *Client) error

	// Delete removes a client
	Delete(client *Client) error

	// FindByID finds a client by ID
	FindByID(clientID id.ULID) (*Client, error)

	// FindByConnectionID finds a client by connection ID
	FindByConnectionID(connID id.ULID) (*Client, error)

	// FindBySubscriptionID finds a client by subscription ID
	FindBySubscriptionID(subID id.ULID) (*Client, error)

	// FindByAlias finds a client by client-provided alias
	FindByAlias(alias string, projID id.Int) (*Client, error)

	// FindAllByProjectID finds all clients for a project
	FindAllByProjectID(projID id.Int) ([]*Client, error)

	// CleanDisconnectedClients removes disconnected clients older than the specified duration
	CleanDisconnectedClients() int
}
