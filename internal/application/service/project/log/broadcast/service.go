package broadcast

import (
	"github.com/qpubio/qpub-server/internal/domain/messaging/publication"
	"github.com/qpubio/qpub-server/internal/domain/project/log"
	broadcastDomain "github.com/qpubio/qpub-server/internal/domain/project/log/broadcast"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	logType "github.com/qpubio/qpub-server/internal/shared/type/log"
)

type Service struct {
	logger             logger.Service
	publicationService publication.Service
}

func NewService(
	logger logger.Service,
	publicationService publication.Service,
) broadcastDomain.Service {
	return &Service{
		logger:             logger,
		publicationService: publicationService,
	}
}

func (s *Service) PublishLog(projectID id.Int, eventType log.EventType, event log.Event) error {
	eventName := string(eventType)

	message := &publication.Message{
		ProjectID:   projectID,
		ChannelName: "_logs",
		Messages: []publication.Payload{{
			Event: &eventName,
			Data:  event,
		}},
	}

	// Publish asynchronously to avoid blocking the main flow
	go func() {
		// Skip stats tracking for internal log broadcasts to avoid affecting project statistics
		if _, err := s.publicationService.Publish("", message, true); err != nil {
			s.logger.Warn(logType.ProjectLog, `Failed to publish log event projectID=%v eventType=%v error=%v`, projectID,
				eventType,
				err)
			return
		}

		s.logger.Debug(logType.ProjectLog, `Published log event projectID=%v channel=%v eventType=%v`, projectID,
			"_logs",
			eventType)
	}()

	return nil
}
