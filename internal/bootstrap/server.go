package bootstrap

import (
	"time"

	"github.com/qpubio/qpub-server/internal/api/http/middleware"
	"github.com/qpubio/qpub-server/internal/api/http/routes/control"
	"github.com/qpubio/qpub-server/internal/api/http/routes/rest"
	"github.com/qpubio/qpub-server/internal/api/http/routes/websocket"
	"github.com/qpubio/qpub-server/internal/bootstrap/container"
	"github.com/qpubio/qpub-server/internal/domain/apikey"
	authAPIKey "github.com/qpubio/qpub-server/internal/domain/auth/apikey"
	"github.com/qpubio/qpub-server/internal/domain/auth/token"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"github.com/gin-gonic/gin"
)

func (a *App) setupHTTPServers() error {
	a.logger.Info(log.App, "Setting up HTTP servers...")

	authTokenService, err := container.GetTyped[token.Service](a.container)
	if err != nil {
		return err
	}
	authApiKeyService, err := container.GetTyped[authAPIKey.Service](a.container)
	if err != nil {
		return err
	}
	apiKeyService, err := container.GetTyped[apikey.Service](a.container)
	if err != nil {
		return err
	}

	controlRouter := gin.Default()
	restRouter := gin.Default()
	websocketRouter := gin.Default()

	controlRouter.Use(middleware.CORS("control", &a.config.Infrastructure.CORS))
	restRouter.Use(middleware.CORS("rest", &a.config.Infrastructure.CORS))
	websocketRouter.Use(middleware.CORS("websocket", &a.config.Infrastructure.CORS))

	controlRouter.Use(middleware.Logger(a.logger))
	restRouter.Use(middleware.Logger(a.logger))
	websocketRouter.Use(middleware.Logger(a.logger))

	control.SetupRoutes(controlRouter, a.config, a.handlers.ControlHandler)

	rest.SetupRoutes(
		restRouter,
		authApiKeyService,
		apiKeyService,
		a.apikeyParser,
		authTokenService,
		*a.token,
		a.handlers.AuthTokenHandler,
		a.handlers.PublicationHandler,
		a.handlers.QueueConfigHandler,
		a.handlers.QueueJobHandler,
		a.handlers.QueuePullHandler,
		a.handlers.QueueWorkerHandler,
	)

	websocket.SetupRoutes(
		websocketRouter,
		*a.token,
		authApiKeyService,
		authTokenService,
		a.handlers.WebsocketHandler,
	)

	go a.startServer(controlRouter, a.config.Infrastructure.Server.ControlPort, "control")
	go a.startServer(restRouter, a.config.Infrastructure.Server.RestPort, "rest")
	go a.startServer(websocketRouter, a.config.Infrastructure.Server.WebSocketPort, "websocket")

	a.logger.Info(log.App, "HTTP servers initialized successfully")
	return nil
}

func (a *App) startServer(router *gin.Engine, port string, serverType string) {
	maxRetries := 3
	var err error
	for i := 0; i < maxRetries; i++ {
		a.logger.Info(log.App, "[%s] Attempting to start server on port %s (attempt %d)...",
			serverType, port, i+1)
		err = router.Run(":" + port)
		if err == nil {
			return
		}
		a.logger.Error(log.App, "[%s] Failed to start server on port %s: %v", serverType, port, err)
		time.Sleep(time.Second)
	}
	a.logger.Error(log.App, "[%s] Exhausted retries starting server on port %s: %v", serverType, port, err)
}
