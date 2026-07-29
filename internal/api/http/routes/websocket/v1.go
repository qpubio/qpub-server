package websocket

import (
	"encoding/json"
	"github.com/qpubio/qpub-server/internal/api/http/middleware/auth"
	"github.com/qpubio/qpub-server/internal/api/websocket/handler"
	"github.com/qpubio/qpub-server/internal/domain/auth/apikey"
	"github.com/qpubio/qpub-server/internal/domain/auth/token"
	"github.com/qpubio/qpub-server/internal/shared/id"
	sharedToken "github.com/qpubio/qpub-server/internal/shared/token"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(
	router *gin.Engine,
	tokenUtil sharedToken.Service,
	authAPIKeyService apikey.Service,
	authTokenService token.Service,
	wsHandler *handler.Handler,
) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	v1 := router.Group("/v1")
	{
		v1.GET(
			"",
			auth.APIKeyOrTokenAuth(authAPIKeyService, authTokenService, tokenUtil),
			func(c *gin.Context) {
				// Get project ID and API key ID and client alias from gin context
				projectID := c.MustGet("projectID").(*id.Int)
				apiKeyID := c.MustGet("apiKeyID").(*id.Int)
				apiPublicKey := c.MustGet("apiPublicKey").(*string)
				alias := c.MustGet("alias").(*string)
				permission := c.MustGet("permission").(*json.RawMessage)

				// Bypass Gin's handling for WebSocket
				wsHandler.HandleConnection(c.Writer, c.Request, projectID, apiKeyID, apiPublicKey, alias, permission)
			})
	}
}
