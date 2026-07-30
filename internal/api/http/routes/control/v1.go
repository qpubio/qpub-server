package control

import (
	"net/http"
	"strings"

	controlHandler "github.com/qpubio/qpub-server/internal/api/http/handler/control"
	"github.com/qpubio/qpub-server/internal/config"

	"github.com/gin-gonic/gin"
)

// SetupRoutes mounts the control API used to provision tenants, keys, limits, and queue admin.
func SetupRoutes(router *gin.Engine, cfg *config.Config, h *controlHandler.Handler) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	if h == nil {
		return
	}

	v1 := router.Group("/control/v1")
	v1.Use(requireControlToken(cfg.App.ControlAPIToken))
	{
		v1.POST("/tenants", h.CreateTenant)
		v1.GET("/tenants/:tenantID", h.GetTenant)
		v1.DELETE("/tenants/:tenantID", h.DeleteTenant)

		v1.PUT("/tenants/:tenantID/limits", h.SetLimits)
		v1.GET("/tenants/:tenantID/limits", h.GetLimits)

		v1.POST("/tenants/:tenantID/keys", h.CreateAPIKey)
		v1.GET("/tenants/:tenantID/keys", h.ListAPIKeys)
		v1.DELETE("/tenants/:tenantID/keys/:keyID", h.DeleteAPIKey)

		v1.GET("/tenants/:tenantID/queues", h.ListQueues)
		v1.GET("/tenants/:tenantID/queues/:queueName", h.GetQueue)
		v1.GET("/tenants/:tenantID/queues/:queueName/jobs", h.ListJobs)
		v1.GET("/tenants/:tenantID/queues/:queueName/jobs/counts", h.GetJobCounts)
		v1.GET("/tenants/:tenantID/queues/:queueName/jobs/:jobId", h.GetJob)
		v1.DELETE("/tenants/:tenantID/queues/:queueName/jobs/:jobId", h.CancelJob)
		v1.POST("/tenants/:tenantID/queues/:queueName/jobs/:jobId/retry", h.RetryJob)
		v1.GET("/tenants/:tenantID/workers", h.ListWorkers)

		v1.GET("/metrics", h.MetricsExport)
	}
}

func requireControlToken(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expected == "" {
			// Dev-friendly: open control API when token unset (non-production local).
			c.Next()
			return
		}
		auth := c.GetHeader("Authorization")
		token := c.GetHeader("X-Control-Token")
		if token == "" && strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
		if token != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
