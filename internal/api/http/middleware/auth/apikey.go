package auth

import (
	"github.com/qpubio/qpub-server/internal/api/response"
	"github.com/qpubio/qpub-server/internal/domain/auth/apikey"
	sharedAPIKey "github.com/qpubio/qpub-server/internal/shared/apikey"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth middleware authenticates requests using API keys
func APIKeyAuth(apiKeyAuth apikey.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !tryAPIKeyAuth(c, apiKeyAuth) {
			response.Unauthorized(c, "Invalid or missing API key")
			c.Abort()
			return
		}
		c.Next()
	}
}

// tryAPIKeyAuth attempts to authenticate using API key without aborting the context
// Returns true if authentication succeeds, false otherwise
func tryAPIKeyAuth(c *gin.Context, apiKeyAuth apikey.Service) bool {
	// Get API key from request (header or query param)
	apiKeyService := sharedAPIKey.NewService()
	apiKeyString, err := apiKeyService.GetKey(c)
	if err != nil {
		return false
	}

	// Authenticate the API key
	apiKey, err := apiKeyAuth.Authenticate(apiKeyString)
	if err != nil {
		return false
	}

	// Set API key ID and project ID in context for future use
	c.Set("apiKeyID", &apiKey.ID)
	c.Set("apiKeyPublicID", &apiKey.PublicID)
	c.Set("apiPublicKey", &apiKey.PublicID)
	c.Set("apiSecretKey", &apiKey.SecretKey)
	c.Set("projectID", &apiKey.ProjectID)

	// Set client alias in context if provided in query params
	alias := c.Query("alias")
	c.Set("alias", &alias)

	// Set API key permission in context
	c.Set("permission", &apiKey.Permission)

	return true
}
