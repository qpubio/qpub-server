package middleware

import (
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger(logger logger.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		t := clock.Now()

		// before request
		logger.Info(log.GinMiddleware, "Starting request to %s", c.Request.URL.Path)

		c.Next()

		// after request
		latency := time.Since(t)
		status := c.Writer.Status()

		logger.Info(log.GinMiddleware, "Completed request to %s in %v with status %d",
			c.Request.URL.Path,
			latency,
			status,
		)
	}
}
