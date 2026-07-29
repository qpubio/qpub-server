package claim

import (
	"encoding/json"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"github.com/golang-jwt/jwt/v4"
)

// UserClaim represents the claims for a user token
type UserClaim struct {
	UserID id.Hash  `json:"uid,omitempty"`
	Roles  []string `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

// APIKeyClaim represents the claims for an API key token
type APIKeyClaim struct {
	Alias      string          `json:"alias,omitempty"`
	Permission json.RawMessage `json:"permission,omitempty"`
	jwt.RegisteredClaims
}

// CreateUserClaimParams is a struct to hold parameters for creating a new UserClaim instance
type CreateUserClaimParams struct {
	UserID id.Hash
	Roles  []string
}

// CreateAPIKeyClaimParams is a struct to hold parameters for creating a new APIKeyClaim instance
type CreateAPIKeyClaimParams struct {
	Alias      string
	Permission json.RawMessage
}

// CreateUserClaim creates a new UserClaim instance with validation
func CreateUserClaim(params CreateUserClaimParams) (*UserClaim, error) {
	// Validate params
	validator := NewValidator()
	if err := validator.ValidateCreateUserClaim(params); err != nil {
		return nil, err
	}

	userClaim := &UserClaim{
		UserID: params.UserID,
		Roles:  params.Roles,
	}

	userClaim.ID = id.NewULID()

	return userClaim, nil
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
