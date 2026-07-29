package publication

import (
	"time"

	"github.com/qpubio/qpub-server/internal/shared/id"
)

// PublishResult carries metadata for a successful broker publication (broadcast envelope).
type PublishResult struct {
	MessageID    id.ULID
	PublishedAt  time.Time
	Channel      string
	PayloadCount int
}

// Message represents a message to be processed by the publisher
type Message struct {
	ProjectID   id.Int
	ChannelName string
	Messages    []Payload
}

type Payload struct {
	Alias *string
	Event *string
	Data  interface{}
}

// CreateParams is a struct to hold parameters for creating a new message
type CreateParams struct {
	ProjectID   id.Int
	ChannelName string
	Messages    []Payload
}

// Create creates a new message instance with validation
func Create(params CreateParams) (*Message, error) {
	validator := NewValidator()
	if err := validator.ValidateCreate(params); err != nil {
		return nil, err
	}

	return &Message{
		ProjectID:   params.ProjectID,
		ChannelName: params.ChannelName,
		Messages:    params.Messages,
	}, nil
}
