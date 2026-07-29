package auth

import (
	"github.com/qpubio/qpub-server/internal/api/response"
	"github.com/qpubio/qpub-server/internal/domain/auth/apikey"
	"github.com/qpubio/qpub-server/internal/domain/auth/token"
	sharedToken "github.com/qpubio/qpub-server/internal/shared/token"

	"github.com/gin-gonic/gin"
)

func APIKeyOrTokenAuth(apiKeyAuth apikey.Service, tokenAuth token.Service, tokenUtil sharedToken.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try API key authentication first
		if tryAPIKeyAuth(c, apiKeyAuth) {
			c.Next()
			return
		}

		// If API key auth failed, try token authentication
		if tryTokenAuth(c, tokenUtil, tokenAuth) {
			c.Next()
			return
		}

		// Both authentication methods failed
		response.Unauthorized(c, "Unauthorized: Invalid API key or token")
		c.Abort()
	}
}
