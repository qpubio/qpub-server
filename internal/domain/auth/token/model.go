package token

import (
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"time"
)

// TokenType represents the type of token
type TokenType string

// TokenType constants
const (
	TokenTypeAPIKey TokenType = "api_key"
)

// RevokedToken represents a revoked token in the database
type RevokedToken struct {
	ID        id.Int    `gorm:"primarykey;autoincrement"`
	TokenID   string    `gorm:"uniqueIndex;not null"` // JWT ID (jti claim)
	OwnerID   id.Int    `gorm:"index"`                // ID of the token owner (User or APIKey)
	OwnerType TokenType `gorm:"type:varchar(10);index"`
	ExpiresAt time.Time `gorm:"index;not null"`
	RevokedAt time.Time
}

// TableName returns the table name for the RevokedToken model
func (RevokedToken) TableName() string {
	return "revoked_tokens"
}

// CreateRevokeParams is a struct to hold parameters for creating a new RevokedToken
type CreateRevokeParams struct {
	TokenID   string
	OwnerID   id.Int
	OwnerType TokenType
	ExpiresAt time.Time
}

// CreateRevoke creates a new RevokedToken instance with validation
func CreateRevoke(params CreateRevokeParams) (*RevokedToken, error) {
	// Validate params
	validator := NewValidator()
	if err := validator.ValidateCreateRevoke(params); err != nil {
		return nil, err
	}

	// Create the RevokedToken instance
	revokedToken := &RevokedToken{
		TokenID:   params.TokenID,
		OwnerID:   params.OwnerID,
		OwnerType: params.OwnerType,
		ExpiresAt: params.ExpiresAt,
		RevokedAt: clock.Now(),
	}

	return revokedToken, nil
}
