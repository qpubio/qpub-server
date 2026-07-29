package publication

import "github.com/qpubio/qpub-server/internal/shared/id"

// Service defines the interface for publisher service operations
type Service interface {
	// Publish publishes a message to a channel.
	// skipStats: if true, does not track message/bandwidth statistics for this publication.
	// On success, returns metadata matching the broadcast DataMessage envelope (id, timestamp).
	Publish(connID id.ULID, message *Message, skipStats bool) (*PublishResult, error)
}
