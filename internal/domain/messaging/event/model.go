package event

import (
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"time"
)

// EventType represents the type of messaging event
type EventType string

// Event type constants for messaging system
const (
	// Connection events
	EventConnectionRegistered   EventType = "connection.registered"
	EventConnectionUnregistered EventType = "connection.unregistered"

	// Client events
	EventClientConnected    EventType = "client.connected"
	EventClientDisconnected EventType = "client.disconnected"

	// Channel events
	EventChannelCreated      EventType = "channel.created"
	EventChannelSubscribed   EventType = "channel.subscribed"
	EventChannelUnsubscribed EventType = "channel.unsubscribed"
	EventChannelEmpty        EventType = "channel.empty"
	EventChannelDeleted      EventType = "channel.deleted"

	// Subscription events
	EventSubscriptionCreated EventType = "subscription.created"
	EventSubscriptionClosed  EventType = "subscription.closed"
)

// Event represents a domain event in the messaging system
type Event struct {
	ID        id.ULID
	Type      EventType
	Timestamp time.Time
	Data      interface{}
}

// NewEvent creates a new event instance
func NewEvent(eventType EventType, data interface{}) *Event {
	return &Event{
		ID:        id.NewULID(),
		Type:      eventType,
		Timestamp: clock.Now(),
		Data:      data,
	}
}

// ConnectionRegisteredData contains data for connection registered event
type ConnectionRegisteredData struct {
	ConnectionID id.ULID
	ProjectID    id.Int
	RemoteAddr   string
	UserAgent    string
}

// ConnectionUnregisteredData contains data for connection unregistered event
type ConnectionUnregisteredData struct {
	ConnectionID id.ULID
	ProjectID    id.Int
}

// ClientConnectedData contains data for client connected event
type ClientConnectedData struct {
	ClientID     id.ULID
	ConnectionID id.ULID
	ProjectID    id.Int
	APIKeyID     id.Int
	UserClientID *string
}

// ClientDisconnectedData contains data for client disconnected event
type ClientDisconnectedData struct {
	ClientID       id.ULID
	ConnectionID   id.ULID
	ProjectID      id.Int
	SubscriptionID *id.ULID
}

// ChannelCreatedData contains data for channel created event
type ChannelCreatedData struct {
	ChannelID   id.ULID
	ChannelName string
	ProjectID   id.Int
	InstanceID  id.ULID
}

// ChannelSubscribedData contains data for channel subscribed event
type ChannelSubscribedData struct {
	ChannelName    string
	ProjectID      id.Int
	ClientID       id.ULID
	SubscriptionID id.ULID
}

// ChannelUnsubscribedData contains data for channel unsubscribed event
type ChannelUnsubscribedData struct {
	ChannelName    string
	ProjectID      id.Int
	ClientID       id.ULID
	SubscriptionID id.ULID
}

// ChannelEmptyData contains data for channel empty event
type ChannelEmptyData struct {
	ChannelName string
	ProjectID   id.Int
	InstanceID  id.ULID
}

// ChannelDeletedData contains data for channel deleted event
type ChannelDeletedData struct {
	ChannelName string
	ProjectID   id.Int
	InstanceID  id.ULID
}

// SubscriptionCreatedData contains data for subscription created event
type SubscriptionCreatedData struct {
	SubscriptionID id.ULID
	ClientID       id.ULID
	ProjectID      id.Int
}

// SubscriptionClosedData contains data for subscription closed event
type SubscriptionClosedData struct {
	SubscriptionID id.ULID
	ClientID       id.ULID
	ProjectID      id.Int
	ChannelCount   int // Number of channels this subscription was subscribed to
}

