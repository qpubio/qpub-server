package rest

import (
	tokenHandler "github.com/qpubio/qpub-server/internal/api/http/handler/auth/token"
	"github.com/qpubio/qpub-server/internal/api/http/handler/messaging/publication"
	queueJobHandler "github.com/qpubio/qpub-server/internal/api/http/handler/queue/job"
	queueHandler "github.com/qpubio/qpub-server/internal/api/http/handler/queue/queue"
	queuePullHandler "github.com/qpubio/qpub-server/internal/api/http/handler/queue/pull"
	queueWorkerHandler "github.com/qpubio/qpub-server/internal/api/http/handler/queue/worker"
	"github.com/qpubio/qpub-server/internal/api/http/middleware/auth"
	authAPIKey "github.com/qpubio/qpub-server/internal/domain/auth/apikey"
	"github.com/qpubio/qpub-server/internal/domain/auth/token"
	projectAPIKey "github.com/qpubio/qpub-server/internal/domain/apikey"
	sharedAPIKey "github.com/qpubio/qpub-server/internal/shared/apikey"
	sharedToken "github.com/qpubio/qpub-server/internal/shared/token"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(
	router *gin.Engine,
	authAPIKeyService authAPIKey.Service,
	projectAPIKeyService projectAPIKey.Service,
	apiKeyParser *sharedAPIKey.Parser,
	authTokenService token.Service,
	tokenUtil sharedToken.Service,
	tokenIssueHandler *tokenHandler.Handler,
	pubHandler *publication.Handler,
	queueConfigHandler *queueHandler.Handler,
	queueJobH *queueJobHandler.Handler,
	queuePullH *queuePullHandler.Handler,
	queueWorkerH *queueWorkerHandler.Handler,
) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	useAPIKeyAuth := auth.APIKeyAuth(authAPIKeyService)
	useTokenRequestAuth := auth.TokenRequestAuth(projectAPIKeyService, apiKeyParser)
	useAuth := auth.APIKeyOrTokenAuth(authAPIKeyService, authTokenService, tokenUtil)

	v1 := router.Group("/v1")
	{
		v1.POST("/key/:keyID/token/issue", useAPIKeyAuth, tokenIssueHandler.IssueToken) // Issue a token for a key
		v1.POST("/key/:keyID/token/request", useTokenRequestAuth, tokenIssueHandler.RequestToken)
		v1.POST("/channel/:channelName/messages", useAuth, pubHandler.Publish) // Publish message(s) to a channel
		v1.POST("/channels/messages", useAuth, pubHandler.Publish)             // Publish message(s) to multiple channels

		// Queue product routes
		v1.PUT("/queue/:queueName", useAuth, queueConfigHandler.Upsert)
		v1.GET("/queue/:queueName", useAuth, queueConfigHandler.Get)
		v1.POST("/queue/:queueName/jobs", useAuth, queueJobH.Enqueue)
		v1.GET("/queue/:queueName/jobs", useAuth, queueJobH.List)
		v1.GET("/queue/:queueName/jobs/:jobId", useAuth, queueJobH.Get)
		v1.DELETE("/queue/:queueName/jobs/:jobId", useAuth, queueJobH.Cancel)
		v1.POST("/queue/:queueName/jobs/:jobId/retry", useAuth, queueJobH.Retry)
		v1.POST("/queue/:queueName/pull", useAuth, queuePullH.Pull)
		v1.POST("/queue/:queueName/jobs/:jobId/ack", useAuth, queuePullH.Ack)
		v1.POST("/queue/:queueName/jobs/:jobId/nack", useAuth, queuePullH.Nack)
		v1.POST("/workers/register", useAuth, queueWorkerH.Register)
		v1.POST("/workers/heartbeat", useAuth, queueWorkerH.Heartbeat)
	}
}
