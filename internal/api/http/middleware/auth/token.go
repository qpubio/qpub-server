package auth

import (
	"github.com/qpubio/qpub-server/internal/api/response"
	authToken "github.com/qpubio/qpub-server/internal/domain/auth/token"
	"github.com/qpubio/qpub-server/internal/shared/permission"
	sharedToken "github.com/qpubio/qpub-server/internal/shared/token"

	"github.com/gin-gonic/gin"
)

func TokenAuth(tokenUtil sharedToken.Service, tokenAuth authToken.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !tryTokenAuth(c, tokenUtil, tokenAuth) {
			response.Unauthorized(c, "Invalid or missing token")
			c.Abort()
			return
		}
		c.Next()
	}
}

// tryTokenAuth attempts to authenticate using token without aborting the context
// Returns true if authentication succeeds, false otherwise
func tryTokenAuth(c *gin.Context, tokenUtil sharedToken.Service, tokenAuth authToken.Service) bool {
	tokenString, err := tokenUtil.GetToken(c)
	if err != nil {
		return false
	}

	apiKey, claims, err := tokenAuth.AuthenticateAPIKeyToken(tokenString)
	if err != nil {
		return false
	}

	// Set API key ID and project ID in context for future use
	c.Set("apiKeyID", &apiKey.ID)
	c.Set("apiKeyPublicID", &apiKey.PublicID)
	c.Set("apiPublicKey", &apiKey.PublicID)
	c.Set("apiSecretKey", &apiKey.SecretKey)
	c.Set("projectID", &apiKey.ProjectID)

	// Set client alias in context if provided in claims
	alias := claims.Alias
	c.Set("alias", &alias)

	// Set permission in context if provided in claims, otherwise set the API key permission
	if permission.HasData(claims.Permission) {
		c.Set("permission", &claims.Permission)
	} else {
		c.Set("permission", &apiKey.Permission)
	}

	return true
}
