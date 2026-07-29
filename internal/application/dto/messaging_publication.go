package dto

import (
	"time"

	domainPublication "github.com/qpubio/qpub-server/internal/domain/messaging/publication"
)

// PublishedChannel describes one successful REST publication (broadcast envelope metadata).
type PublishedChannel struct {
	Channel      string    `json:"channel"`
	MessageID    string    `json:"message_id"`
	PublishedAt  time.Time `json:"published_at"`
	PayloadCount int       `json:"payload_count"`
}

// PublishMessageResponse is the JSON body for POST /v1/channel/.../messages and /v1/channels/messages on success.
type PublishMessageResponse struct {
	Published []PublishedChannel `json:"published"`
}

// NewPublishMessageResponse maps domain publish results to the REST response DTO.
func NewPublishMessageResponse(results []domainPublication.PublishResult) PublishMessageResponse {
	out := make([]PublishedChannel, len(results))
	for i, r := range results {
		out[i] = PublishedChannel{
			Channel:      r.Channel,
			MessageID:    string(r.MessageID),
			PublishedAt:  r.PublishedAt,
			PayloadCount: r.PayloadCount,
		}
	}
	return PublishMessageResponse{Published: out}
}
