package token

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// Service provides token operations
type Service struct {
	decoder *Decoder
}

// NewService creates a new token service
func NewService() *Service {
	return &Service{
		decoder: NewDecoder(),
	}
}

// GetToken attempts to get token from header then query params
func (s *Service) GetToken(c *gin.Context) (string, error) {
	if token, err := s.GetTokenFromHeader(c); err == nil {
		return token, nil
	}
	return s.GetTokenFromQuery(c)
}

// GetTokenFromHeader extracts token from Authorization header
func (s *Service) GetTokenFromHeader(c *gin.Context) (string, error) {
	auth := c.GetHeader("Authorization")
	if auth == "" || !strings.HasPrefix(auth, bearerPrefix) {
		return "", ErrInvalidToken
	}
	return auth[len(bearerPrefix):], nil
}

// GetTokenFromQuery extracts token from query parameter
func (s *Service) GetTokenFromQuery(c *gin.Context) (string, error) {
	token := c.Query("access_token")
	if token == "" {
		return "", ErrInvalidToken
	}
	return s.decoder.DecodeToken(token)
}
