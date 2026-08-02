package claim

import (
	"encoding/json"

	"github.com/qpubio/qpub-server/internal/shared/id"

	"github.com/golang-jwt/jwt/v4"
)

// APIKeyClaim represents the claims for an API key token
type APIKeyClaim struct {
	Alias      string          `json:"alias,omitempty"`
	Permission json.RawMessage `json:"permission,omitempty"`
	jwt.RegisteredClaims
}

// CreateAPIKeyClaimParams is a struct to hold parameters for creating a new APIKeyClaim instance
type CreateAPIKeyClaimParams struct {
	Alias      string
	Permission json.RawMessage
}

// CreateAPIKeyClaim creates a new APIKeyClaim instance with validation
func CreateAPIKeyClaim(params CreateAPIKeyClaimParams) (*APIKeyClaim, error) {
	// Validate params
	validator := NewValidator()
	if err := validator.ValidateCreateAPIKeyClaim(params); err != nil {
		return nil, err
	}

	apiKeyClaim := &APIKeyClaim{
		Alias:      params.Alias,
		Permission: params.Permission,
	}

	apiKeyClaim.ID = id.NewULID()

	return apiKeyClaim, nil
}
