package apikey

import "github.com/qpubio/qpub-server/internal/shared/id"

// Repository defines the interface for API key persistence operations
type Repository interface {
	// Create persists a new API key to the storage
	Create(apiKey *APIKey) (id.Int, error)

	// Update modifies an existing API key in the storage
	Update(apiKey *APIKey) error

	// Delete deletes an API key from the storage
	Delete(apiKey *APIKey) error

	// FindByID retrieves an API key by its ID
	FindByID(akID id.Int) (*APIKey, error)

	// FindByIDs retrieves API keys by their IDs
	FindByIDs(akIDs []id.Int) ([]APIKey, error)

	// ListByProjectID retrieves API keys by their project ID
	ListByProjectID(projID id.Int) ([]APIKey, error)
}
