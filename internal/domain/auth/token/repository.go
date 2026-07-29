package token

import "github.com/qpubio/qpub-server/internal/shared/id"

// Repository defines the interface for token persistence operations
type Repository interface {
	// CreateRevoke persists a new revoked token to the storage
	CreateRevoke(token RevokedToken) (id.Int, error)

	// PurgeExpired purges all expired revoked tokens from the storage
	PurgeExpired() error

	// FindByTokenID retrieves a revoked token by its token ID
	FindByTokenID(tokenID id.ULID) (RevokedToken, error)
}
