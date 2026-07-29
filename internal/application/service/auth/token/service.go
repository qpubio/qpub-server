package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qpubio/qpub-server/internal/config"
	"github.com/qpubio/qpub-server/internal/domain/apikey"
	"github.com/qpubio/qpub-server/internal/domain/auth/token"
	"github.com/qpubio/qpub-server/internal/domain/auth/token/claim"
	sharedAPIKey "github.com/qpubio/qpub-server/internal/shared/apikey"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

type Service struct {
	config        *config.Config
	repository    token.Repository
	apiKeyService apikey.Service
	apiKeyParser  *sharedAPIKey.Parser
}

func NewService(
	config *config.Config,
	repository token.Repository,
	apiKeyService apikey.Service,
	apiKeyParser *sharedAPIKey.Parser,
) token.Service {
	return &Service{
		config:        config,
		repository:    repository,
		apiKeyService: apiKeyService,
		apiKeyParser:  apiKeyParser,
	}
}

func (s *Service) CreateAPIKeyToken(apiKeyPublicID string, permission json.RawMessage, alias string, secretKey []byte, expiresIn time.Duration) (string, error) {
	if len(secretKey) == 0 {
		return "", fmt.Errorf("missing secret key")
	}

	params := claim.CreateAPIKeyClaimParams{
		Alias:      alias,
		Permission: permission,
	}

	apiKeyClaim, err := claim.CreateAPIKeyClaim(params)
	if err != nil {
		return "", fmt.Errorf("failed to create API key claim: %w", err)
	}

	now := clock.Now()
	apiKeyClaim.IssuedAt = jwt.NewNumericDate(now)
	apiKeyClaim.ExpiresAt = jwt.NewNumericDate(now.Add(expiresIn))

	jwtToken := jwt.NewWithClaims(jwt.GetSigningMethod(s.config.Auth.Token.Signing.Method), apiKeyClaim)
	jwtToken.Header["aki"] = apiKeyPublicID

	return jwtToken.SignedString(secretKey)
}

func (s *Service) VerifyAPIKeyToken(tokenString string, secretKey []byte) (*claim.APIKeyClaim, error) {
	if len(secretKey) == 0 {
		return nil, fmt.Errorf("missing secret key")
	}

	jwtToken, err := jwt.ParseWithClaims(tokenString, &claim.APIKeyClaim{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != s.config.Auth.Token.Signing.Method {
			return nil, fmt.Errorf("invalid token signature: unexpected signing method: %v", t.Header["alg"])
		}
		return secretKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("token has expired")
		}
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	claims, ok := jwtToken.Claims.(*claim.APIKeyClaim)
	if !ok || !jwtToken.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	if claims.ID != "" {
		_, err = s.repository.FindByTokenID(claims.ID)
		if err == nil {
			return nil, fmt.Errorf("invalid token")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to check token revocation: %w", err)
		}
	}

	return claims, nil
}

func (s *Service) AuthenticateAPIKeyToken(tokenString string) (*apikey.APIKey, *claim.APIKeyClaim, error) {
	header, err := s.DecodeTokenHeader(tokenString)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode token header: %w", err)
	}

	akiValue, ok := header["aki"]
	if !ok || akiValue == nil {
		return nil, nil, fmt.Errorf("missing API key ID in token header")
	}

	aki, ok := akiValue.(string)
	if !ok || aki == "" {
		return nil, nil, fmt.Errorf("invalid API key ID format in token header")
	}

	apiKeyID, err := s.apiKeyParser.ParseComponents(aki)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse API key components: %w", err)
	}

	apiKey, err := s.apiKeyService.Get(apiKeyID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get API key: %w", err)
	}

	claims, err := s.VerifyAPIKeyToken(tokenString, []byte(apiKey.SecretKey))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify API key token: %w", err)
	}

	return &apiKey, claims, nil
}

func (s *Service) DecodeTokenHeader(tokenString string) (map[string]interface{}, error) {
	jwtToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("dummy-key"), nil
	})
	if err != nil {
		if !errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, fmt.Errorf("failed to parse token: %w", err)
		}
	}
	if jwtToken == nil || jwtToken.Header == nil {
		return nil, errors.New("invalid token header")
	}
	return jwtToken.Header, nil
}

func (s *Service) DecodeAPIKeyTokenClaims(tokenString string) (*claim.APIKeyClaim, error) {
	jwtToken, err := jwt.ParseWithClaims(tokenString, &claim.APIKeyClaim{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("dummy-key"), nil
	})
	if err != nil {
		if !errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, fmt.Errorf("failed to parse token: %w", err)
		}
	}
	if claims, ok := jwtToken.Claims.(*claim.APIKeyClaim); ok {
		return claims, nil
	}
	return nil, errors.New("invalid token claims")
}

func (s *Service) RevokeAPIKeyToken(tokenString string, secretKey []byte) error {
	claims, err := s.VerifyAPIKeyToken(tokenString, secretKey)
	if err != nil {
		return fmt.Errorf("failed to verify API key token: %w", err)
	}
	if claims.ID == "" {
		return fmt.Errorf("token has no ID to revoke")
	}
	expiresAt := time.Time{}
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}
	revoked, err := token.CreateRevoke(token.CreateRevokeParams{
		TokenID:   claims.ID,
		OwnerType: token.TokenTypeAPIKey,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return err
	}
	_, err = s.repository.CreateRevoke(*revoked)
	return err
}

func (s *Service) PurgeExpired() error {
	return s.repository.PurgeExpired()
}

func (s *Service) GetByTokenID(tokenID id.ULID) (token.RevokedToken, error) {
	return s.repository.FindByTokenID(tokenID)
}
