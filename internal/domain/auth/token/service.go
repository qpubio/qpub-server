package token

import (
	"encoding/json"
	"github.com/qpubio/qpub-server/internal/domain/apikey"
	"github.com/qpubio/qpub-server/internal/domain/auth/token/claim"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"time"
)

// Service defines client (API-key) token operations for the data plane.
type Service interface {
	CreateAPIKeyToken(apiKeyPublicID string, permission json.RawMessage, alias string, secretKey []byte, expiresIn time.Duration) (string, error)
	VerifyAPIKeyToken(tokenString string, secretKey []byte) (*claim.APIKeyClaim, error)
	AuthenticateAPIKeyToken(tokenString string) (*apikey.APIKey, *claim.APIKeyClaim, error)
	DecodeTokenHeader(tokenString string) (map[string]interface{}, error)
	DecodeAPIKeyTokenClaims(tokenString string) (*claim.APIKeyClaim, error)
	RevokeAPIKeyToken(tokenString string, secretKey []byte) error
	PurgeExpired() error
	GetByTokenID(tokenID id.ULID) (RevokedToken, error)
}
