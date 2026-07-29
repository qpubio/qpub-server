package modules

import (
	broadcastService "github.com/qpubio/qpub-server/internal/application/service/project/log/broadcast"
	"github.com/qpubio/qpub-server/internal/domain/messaging/publication"
	broadcastDomain "github.com/qpubio/qpub-server/internal/domain/project/log/broadcast"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
)

func NewProjectLogBroadcastModule(
	logger logger.Service,
	publicationService publication.Service,
) broadcastDomain.Service {
	return broadcastService.NewService(logger, publicationService)
}
