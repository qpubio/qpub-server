package apikey

import (
	"github.com/qpubio/qpub-server/internal/shared/security"
)

// Security handles API key security operations
type Security struct {
	secret *security.Secret
}

// NewSecurity creates a new Security instance
func NewSecurity() *Security {
	return &Security{
		secret: security.NewSecret(),
	}
}

// GenerateSecretKey generates a secret key
func (s *Security) GenerateSecretKey() (string, error) {
	secretKey, err := s.secret.Generate(64)
	if err != nil {
		return "", wrapError(ErrSecurityInit, "failed to generate secret key")
	}
	return secretKey, nil
}
