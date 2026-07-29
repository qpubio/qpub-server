package broadcast

import (
	"github.com/qpubio/qpub-server/internal/domain/project/log"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Service defines the interface for broadcasting project log events
type Service interface {
	// PublishLog publishes log event data to the logs channel
	// projectID is used for routing the message but not included in the event payload
	// eventType is used for the protocol message event field
	PublishLog(projectID id.Int, eventType log.EventType, event log.Event) error
}
