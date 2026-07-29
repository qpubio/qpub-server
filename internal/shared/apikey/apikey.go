package apikey

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	basicPrefix = "Basic "
)

// Service provides API key operations
type Service struct {
	decoder *Decoder
	parser  *Parser
}

// NewService creates a new API key service
func NewService() *Service {
	return &Service{
		decoder: NewDecoder(),
		parser:  NewParser(),
	}
}

// GetKey attempts to get API key from header then query params
func (s *Service) GetKey(c *gin.Context) (string, error) {
	if apiKey, err := s.GetKeyFromHeader(c); err == nil {
		return apiKey, nil
	}
	return s.GetKeyFromQuery(c)
}

// GetKeyFromHeader extracts API key from Authorization header
func (s *Service) GetKeyFromHeader(c *gin.Context) (string, error) {
	auth := c.GetHeader("Authorization")
	if auth == "" || !strings.HasPrefix(auth, basicPrefix) {
		return "", ErrInvalidAPIKey
	}
	return s.decoder.DecodeKey(auth[len(basicPrefix):], false)
}

// GetKeyFromQuery extracts API key from query parameter
func (s *Service) GetKeyFromQuery(c *gin.Context) (string, error) {
	apiKey := c.Query("api_key")
	if apiKey == "" {
		return "", ErrInvalidAPIKey
	}
	return s.decoder.DecodeKey(apiKey, true)
}
