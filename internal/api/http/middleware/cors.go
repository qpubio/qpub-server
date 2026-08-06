package middleware

import (
	"strings"

	"github.com/qpubio/qpub-server/internal/config/infrastructure"

	"github.com/gin-gonic/gin"
)

func CORS(routeType string, corsConfig *infrastructure.CORS) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get allowed origins from configuration based on route type
		var allowedOrigins string
		switch routeType {
		case "control":
			allowedOrigins = corsConfig.Control
		case "rest":
			allowedOrigins = corsConfig.Rest
		case "websocket":
			allowedOrigins = corsConfig.WebSocket
		default:
			allowedOrigins = "*"
		}

		origin := c.Request.Header.Get("Origin")

		// Determine if this route type requires credentials
		credentialsRequired := false
		switch routeType {
		case "control":
			credentialsRequired = true
		default:
			credentialsRequired = false
		}

		// Set Access-Control-Allow-Origin safely based on config and route needs
		originMatched := false
		if allowedOrigins == "*" {
			if credentialsRequired && origin != "" {
				// Echo the requesting origin when credentials are required
				c.Header("Access-Control-Allow-Origin", origin)
				originMatched = true
			} else {
				// No credentials required: wildcard is safe
				c.Header("Access-Control-Allow-Origin", "*")
			}
		} else {
			allowedOriginsList := strings.Split(allowedOrigins, ",")
			for _, allowedOrigin := range allowedOriginsList {
				if strings.TrimSpace(allowedOrigin) == origin {
					c.Header("Access-Control-Allow-Origin", origin)
					originMatched = true
					break
				}
			}
		}

		// Help caches/CDNs differentiate responses by Origin when echoing
		if originMatched {
			c.Header("Vary", "Origin")
		}

		// Set standard CORS headers
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		// Only allow credentials when we returned a specific origin (not *) and the route requires it
		if credentialsRequired && originMatched {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Control API accepts Authorization Bearer or X-Control-Token
		if routeType == "control" {
			currentAllowHeaders := c.Writer.Header().Get("Access-Control-Allow-Headers")
			c.Header("Access-Control-Allow-Headers", currentAllowHeaders+", X-Control-Token")
		}

		// Additional headers specifically for WebSocket
		if routeType == "websocket" {
			// Allow WebSocket protocol headers
			currentAllowHeaders := c.Writer.Header().Get("Access-Control-Allow-Headers")
			c.Header("Access-Control-Allow-Headers", currentAllowHeaders+", Sec-WebSocket-Key, Sec-WebSocket-Protocol, Sec-WebSocket-Version, Sec-WebSocket-Extensions")

			// Required for WebSocket protocol upgrade
			c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
