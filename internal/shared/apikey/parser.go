package apikey

import (
	"github.com/qpubio/qpub-server/internal/shared/id"
	"strings"
)

// Parser handles API key parsing operations
type Parser struct{}

// NewParser creates a new Parser instance
func NewParser() *Parser {
	return &Parser{}
}

// ParseComponents extracts API key ID from the public API key identifier.
func (p *Parser) ParseComponents(apiKeyPublicID string) (apiKeyID id.Int, err error) {
	apiKeyID, err = id.ParseHashID(apiKeyPublicID)
	if err != nil {
		return 0, wrapError(err, "invalid API key ID")
	}

	return apiKeyID, nil
}

// ParseKey extracts API key ID and secret key from a full API key.
func (p *Parser) ParseKey(apiKey string) (apiKeyID id.Int, secretKey string, err error) {
	parts := strings.Split(apiKey, ":")
	if len(parts) != 2 {
		return 0, "", ErrInvalidFormat
	}

	apiKeyID, err = p.ParseComponents(parts[0])
	if err != nil {
		return 0, "", wrapError(err, "failed to parse components")
	}

	return apiKeyID, parts[1], nil
}
