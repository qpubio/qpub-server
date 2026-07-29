package broker

import "context"

// MessageHandler is a function that handles received messages
type MessageHandler func(channelName string, message []byte)

// Service defines the broker interaction contract
type Service interface {
	// ListenToChannel registers a handler for messages on a channel from the broker
	ListenToChannel(channelName string, handler MessageHandler) error

	// StopListeningToChannel removes a handler for messages on a channel
	StopListeningToChannel(channelName string) error

	// PublishToChannel publishes a message to a channel via the broker
	PublishToChannel(channelName string, message []byte) error

	// Shutdown gracefully closes all broker connections
	Shutdown(ctx context.Context) error
}
