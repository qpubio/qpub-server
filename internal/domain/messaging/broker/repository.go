package broker

import (
	"context"
)

// Repository defines the data access contract for broker operations
type Repository interface {
	// Subscribe registers a handler for messages on a channel
	Subscribe(channelName string, handler MessageHandler) error

	// Unsubscribe removes a handler for messages on a channel
	Unsubscribe(channelName string) error

	// Publish sends a message to a channel
	Publish(channelName string, message []byte) error

	// Close gracefully closes all broker connections
	Close(ctx context.Context) error
}
