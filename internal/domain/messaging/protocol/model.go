package protocol

import (
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"time"
)

// ActionType defines the type of action
type ActionType int

// Action types
const (
	// Connection actions
	ActionConnect      ActionType = 0
	ActionConnected    ActionType = 1
	ActionDisconnect   ActionType = 2
	ActionDisconnected ActionType = 3

	// Channel actions
	ActionSubscribe    ActionType = 4
	ActionSubscribed   ActionType = 5
	ActionUnsubscribe  ActionType = 6
	ActionUnsubscribed ActionType = 7

	// Data Message actions
	ActionPublish   ActionType = 8
	ActionPublished ActionType = 9
	ActionMessage   ActionType = 10

	// Error actions
	ActionError ActionType = 11

	// Ping/Pong actions
	ActionPing ActionType = 12
	ActionPong ActionType = 13
)

// ActionStrings is a map of ActionType to string for logging or external systems
var ActionStrings = map[ActionType]string{
	ActionConnect:      "connect",
	ActionConnected:    "connected",
	ActionDisconnect:   "disconnect",
	ActionDisconnected: "disconnected",
	ActionSubscribe:    "subscribe",
	ActionSubscribed:   "subscribed",
	ActionUnsubscribe:  "unsubscribe",
	ActionUnsubscribed: "unsubscribed",
	ActionPublish:      "publish",
	ActionPublished:    "published",
	ActionMessage:      "message",
	ActionError:        "error",
	ActionPing:         "ping",
	ActionPong:         "pong",
}

// ActionToString converts an ActionType to a string
func ActionToString(action ActionType) string {
	return ActionStrings[action]
}

// Message is the base protocol entity for all messages
type Message struct {
	Action ActionType `json:"action"`
	Error  *ErrorInfo `json:"error,omitempty"`
}

// NewMessage creates a new base protocol message
func NewMessage(action ActionType, error *ErrorInfo) *Message {
	return &Message{
		Action: action,
		Error:  error,
	}
}

// ConnectionMessage represents a connection state change message
type ConnectionMessage struct {
	Message
	ConnectionID id.ULID           `json:"connection_id,omitempty"`
	Details      ConnectionDetails `json:"connection_details,omitempty"`
}

// ConnectionDetails represents the details of a connection
type ConnectionDetails struct {
	Alias    string `json:"alias"`
	ClientID string `json:"client_id"`
	ServerID string `json:"server_id"`
}

// NewConnectionMessage creates a connection state message
func NewConnectionMessage(
	action ActionType,
	connectionID id.ULID,
	clientID *id.ULID,
	serverID *string,
	alias *string,
	error *ErrorInfo,
) *ConnectionMessage {
	details := ConnectionDetails{}

	// Only set Alias if not nil
	if alias != nil {
		details.Alias = *alias
	}

	// Only set ClientID if not nil
	if clientID != nil {
		details.ClientID = *clientID
	}

	// Only set ServerID if not nil
	if serverID != nil {
		details.ServerID = *serverID
	}

	return &ConnectionMessage{
		Message:      *NewMessage(action, error),
		ConnectionID: connectionID,
		Details:      details,
	}
}

// ChannelMessage represents channel state change message
type ChannelMessage struct {
	Message
	Channel        string  `json:"channel"`
	SubscriptionID id.ULID `json:"subscription_id,omitempty"`
}

// NewChannelMessage creates a channel state message
func NewChannelMessage(
	action ActionType,
	channel string,
	subscriptionID id.ULID,
	error *ErrorInfo,
) *ChannelMessage {
	return &ChannelMessage{
		Message:        *NewMessage(action, error),
		Channel:        channel,
		SubscriptionID: subscriptionID,
	}
}

// DataMessage represents application data being sent
type DataMessage struct {
	Message
	ID        id.ULID              `json:"id,omitempty"`
	Timestamp time.Time            `json:"timestamp,omitempty"`
	Channel   string               `json:"channel"`
	Messages  []DataMessagePayload `json:"messages,omitempty"`
	Error     *ErrorInfo           `json:"error,omitempty"`
}

// NewDataMessage creates a data message
func NewDataMessage(
	action ActionType,
	channel string,
	messages []DataMessagePayload,
	error *ErrorInfo,
) *DataMessage {
	return &DataMessage{
		Message:   *NewMessage(action, error),
		ID:        id.NewULID(),
		Timestamp: clock.Now(),
		Channel:   channel,
		Messages:  messages,
	}
}

// NewDataMessageWithEnvelope builds a DataMessage with explicit id and timestamp.
// Used for publish acknowledgements (ActionPublished) so producers see the same envelope identifiers as subscribers on ActionMessage (broadcast).
func NewDataMessageWithEnvelope(
	action ActionType,
	channel string,
	messages []DataMessagePayload,
	error *ErrorInfo,
	messageID id.ULID,
	timestamp time.Time,
) *DataMessage {
	return &DataMessage{
		Message:   *NewMessage(action, error),
		ID:        messageID,
		Timestamp: timestamp,
		Channel:   channel,
		Messages:  messages,
	}
}

type DataMessagePayload struct {
	Alias *string     `json:"alias,omitempty"`
	Event *string     `json:"event,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

func NewDataMessagePayload(
	alias *string,
	event *string,
	data interface{},
) *DataMessagePayload {
	return &DataMessagePayload{
		Alias: alias,
		Event: event,
		Data:  data,
	}
}

type RESTPublishMessage struct {
	Channels []*string `json:"channels,omitempty"`
	Messages []struct {
		Alias *string     `json:"alias,omitempty"`
		Event *string     `json:"event,omitempty"`
		Data  interface{} `json:"data,omitempty"`
	} `json:"messages,omitempty"`
}

// PingMessage represents a ping message for keep-alive
type PingMessage struct {
	Message
	ID int `json:"id"` // Integer ID of the ping message
}

// NewPingMessage creates a new ping message
func NewPingMessage(ID int) *PingMessage {
	return &PingMessage{
		Message: *NewMessage(ActionPing, nil),
		ID:      ID,
	}
}

// PongMessage represents a pong response message
type PongMessage struct {
	Message
	ID int `json:"id"` // Integer ID of the pong message
}

// NewPongMessage creates a new pong message with the given ID
func NewPongMessage(ID int) *PongMessage {
	return &PongMessage{
		Message: *NewMessage(ActionPong, nil),
		ID:      ID,
	}
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Code       int    `json:"code"`
	Href       string `json:"href"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

// NewErrorInfo creates error information
func NewErrorInfo(
	code int,
	href string,
	message string,
	statusCode int,
) *ErrorInfo {
	return &ErrorInfo{
		Code:       code,
		Href:       href,
		Message:    message,
		StatusCode: statusCode,
	}
}
