package bootstrap

import (
	authTokenHandler "github.com/qpubio/qpub-server/internal/api/http/handler/auth/token"
	controlHandler "github.com/qpubio/qpub-server/internal/api/http/handler/control"
	publicationHandler "github.com/qpubio/qpub-server/internal/api/http/handler/messaging/publication"
	queueJobHandler "github.com/qpubio/qpub-server/internal/api/http/handler/queue/job"
	queuePullHandler "github.com/qpubio/qpub-server/internal/api/http/handler/queue/pull"
	queueConfigHandler "github.com/qpubio/qpub-server/internal/api/http/handler/queue/queue"
	queueWorkerHandler "github.com/qpubio/qpub-server/internal/api/http/handler/queue/worker"
	websocketHandler "github.com/qpubio/qpub-server/internal/api/websocket/handler"
	sessionService "github.com/qpubio/qpub-server/internal/application/service/messaging/session"
	"github.com/qpubio/qpub-server/internal/bootstrap/container"
	"github.com/qpubio/qpub-server/internal/domain/apikey"
	authToken "github.com/qpubio/qpub-server/internal/domain/auth/token"
	"github.com/qpubio/qpub-server/internal/domain/messaging/client"
	"github.com/qpubio/qpub-server/internal/domain/messaging/connection"
	"github.com/qpubio/qpub-server/internal/domain/messaging/publication"
	"github.com/qpubio/qpub-server/internal/domain/messaging/subscription"
	logBroadcast "github.com/qpubio/qpub-server/internal/domain/project/log/broadcast"
	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
	domainRouter "github.com/qpubio/qpub-server/internal/domain/queue/router"
	domainWorker "github.com/qpubio/qpub-server/internal/domain/queue/worker"
	"github.com/qpubio/qpub-server/internal/domain/tenant"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

// HandlerContainer stores data-plane HTTP handlers.
type HandlerContainer struct {
	AuthTokenHandler   *authTokenHandler.Handler
	PublicationHandler *publicationHandler.Handler
	WebsocketHandler   *websocketHandler.Handler
	QueueConfigHandler *queueConfigHandler.Handler
	QueueJobHandler    *queueJobHandler.Handler
	QueuePullHandler   *queuePullHandler.Handler
	QueueWorkerHandler *queueWorkerHandler.Handler
	ControlHandler     *controlHandler.Handler
}

func (a *App) setupHandlers() error {
	a.logger.Info(log.App, "Setting up HTTP handlers...")
	a.handlers = &HandlerContainer{}

	authTokenService, err := container.GetTyped[authToken.Service](a.container)
	if err != nil {
		return err
	}
	a.handlers.AuthTokenHandler = authTokenHandler.NewHandler(authTokenService, a.logger, a.config)

	publicationService, err := container.GetTyped[publication.Service](a.container)
	if err != nil {
		return err
	}
	a.handlers.PublicationHandler = publicationHandler.NewHandler(a.logger, publicationService, a.permission)

	connectionService, err := container.GetTyped[connection.Service](a.container)
	if err != nil {
		return err
	}
	clientService, err := container.GetTyped[client.Service](a.container)
	if err != nil {
		return err
	}
	subscriptionService, err := container.GetTyped[subscription.Service](a.container)
	if err != nil {
		return err
	}
	sessionSvc, err := container.GetTyped[*sessionService.Service](a.container)
	if err != nil {
		return err
	}
	logBroadcaster, err := container.GetTyped[logBroadcast.Service](a.container)
	if err != nil {
		return err
	}

	a.handlers.WebsocketHandler = websocketHandler.NewHandler(
		a.config,
		a.logger,
		a.instanceID,
		a.permission,
		connectionService,
		clientService,
		subscriptionService,
		sessionSvc,
		publicationService,
		logBroadcaster,
	)

	queueRouter, err := container.GetTyped[domainRouter.Service](a.container)
	if err != nil {
		return err
	}
	queueService, err := container.GetTyped[domainQueue.Service](a.container)
	if err != nil {
		return err
	}
	queueJobService, err := container.GetTyped[domainJob.Service](a.container)
	if err != nil {
		return err
	}
	queueWorkerService, err := container.GetTyped[domainWorker.Service](a.container)
	if err != nil {
		return err
	}

	a.handlers.QueueConfigHandler = queueConfigHandler.NewHandler(a.logger, queueService, a.permission)
	a.handlers.QueueJobHandler = queueJobHandler.NewHandler(a.logger, queueRouter, queueJobService, queueService, a.permission)
	a.handlers.QueuePullHandler = queuePullHandler.NewHandler(a.logger, queueRouter, a.permission)
	a.handlers.QueueWorkerHandler = queueWorkerHandler.NewHandler(a.logger, queueWorkerService)

	tenantService, err := container.GetTyped[tenant.Service](a.container)
	if err != nil {
		return err
	}
	apiKeyService, err := container.GetTyped[apikey.Service](a.container)
	if err != nil {
		return err
	}
	a.handlers.ControlHandler = controlHandler.NewHandler(
		tenantService,
		apiKeyService,
		queueService,
		queueWorkerService,
	)

	a.logger.Info(log.App, "HTTP handlers setup completed")
	return nil
}
