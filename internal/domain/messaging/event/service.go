package event

import "context"

// EventHandler is a function that processes events
type EventHandler func(event *Event) error

// HandlerRegistration represents a registered event handler
type HandlerRegistration struct {
	ID          string
	EventType   EventType
	Handler     EventHandler
	Description string
}

// Service defines the event bus service interface
type Service interface {
	// Publish publishes an event to all registered handlers
	Publish(ctx context.Context, event *Event) error

	// PublishSync publishes an event synchronously to all handlers
	PublishSync(ctx context.Context, event *Event) error

	// Subscribe registers a handler for a specific event type
	Subscribe(eventType EventType, handlerID string, handler EventHandler) error

	// SubscribeWithDescription registers a handler with a description
	SubscribeWithDescription(eventType EventType, handlerID string, handler EventHandler, description string) error

	// Unsubscribe removes a handler for a specific event type
	Unsubscribe(eventType EventType, handlerID string) error

	// GetHandlers returns all registered handlers for an event type
	GetHandlers(eventType EventType) []HandlerRegistration

	// GetAllHandlers returns all registered handlers
	GetAllHandlers() map[EventType][]HandlerRegistration

	// Close gracefully shuts down the event bus
	Close(ctx context.Context) error
}
