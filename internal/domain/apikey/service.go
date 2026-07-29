package apikey

import "github.com/qpubio/qpub-server/internal/shared/id"

// Service defines the interface for API key service operations
type Service interface {
	// Create creates a new API key
	Create(params CreateParams) (APIKey, error)

	// Update updates an API key
	Update(akID id.Int, params UpdateParams) (APIKey, error)

	// Delete deletes an API key
	Delete(akID id.Int) error

	// Get retrieves an API key by its ID
	Get(akID id.Int) (APIKey, error)

	// GetByIDs retrieves API keys by their IDs
	GetByIDs(akIDs []id.Int) ([]APIKey, error)

	// ListByProjectID retrieves an API key by its project ID
	ListByProjectID(projID id.Int) ([]APIKey, error)
}
