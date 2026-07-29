package channel

import "github.com/qpubio/qpub-server/internal/shared/id"

// Service is the service for the channel domain
type Service interface {
	// Create creates a new channel
	Create(rawName string, projID id.Int) (*Channel, error)

	// Update updates a channel
	Update(ch *Channel) error

	// Delete removes a channel
	Delete(rawName string, projID id.Int) error

	// GetOrCreate gets a channel by ID, creating it if it doesn't exist
	GetOrCreate(rawName string, projID id.Int) (*Channel, error)

	// Get gets a channel by ID
	Get(rawName string, projID id.Int) (*Channel, error)
}
