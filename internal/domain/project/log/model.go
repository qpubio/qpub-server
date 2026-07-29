package log

import (
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// EventType represents the type of log event
type EventType string

// EventType constants for connection lifecycle and queue activity events
const (
	EventConnectionOpened   EventType = "connection.opened"
	EventConnectionClosed   EventType = "connection.closed"
	EventConnectionError    EventType = "connection.error"
	EventClientConnected    EventType = "client.connected"
	EventClientDisconnected EventType = "client.disconnected"
	EventSubscribed         EventType = "subscription.created"
	EventUnsubscribed       EventType = "subscription.closed"

	EventQueueJobEnqueued  EventType = "queue.job.enqueued"
	EventQueueJobClaimed   EventType = "queue.job.claimed"
	EventQueueJobCompleted EventType = "queue.job.completed"
	EventQueueJobNacked    EventType = "queue.job.nacked"
	EventQueueJobRetried   EventType = "queue.job.retried"
	EventQueueJobDLQ       EventType = "queue.job.dlq"
	EventQueueJobCancelled EventType = "queue.job.cancelled"
	EventQueueWorkerRegistered EventType = "queue.worker.registered"
)

// ConnectionDetails contains connection-related identifiers
type ConnectionDetails struct {
	ConnectionID *id.ULID `json:"connection_id,omitempty"`
	ClientID     *id.ULID `json:"client_id,omitempty"`
	Alias        *string  `json:"alias,omitempty"`
	ChannelName  *string  `json:"channel,omitempty"`
	APIKey       *string  `json:"api_key,omitempty"`
}

// SourceDetails contains information about the connection source
type SourceDetails struct {
	RemoteAddr string  `json:"remote_addr,omitempty"`
	UserAgent  *string `json:"user_agent,omitempty"`
	Origin     *string `json:"origin,omitempty"`
}

// InstanceDetails contains information about the server instance handling the connection
type InstanceDetails struct {
	Site string `json:"site"` // Format: "{regionID}-{zoneID}" e.g. "us-east-1a"
}

// ErrorDetails contains error information when applicable
type ErrorDetails struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// QueueDetails contains queue/job identifiers for queue activity events
type QueueDetails struct {
	QueueName  string   `json:"queue_name,omitempty"`
	JobID      *id.ULID `json:"job_id,omitempty"`
	Status     string   `json:"status,omitempty"`
	WorkerID   string   `json:"worker_id,omitempty"`
	WorkerName string   `json:"worker_name,omitempty"`
	Attempt    int      `json:"attempt,omitempty"`
}

// Event represents a log event for connection lifecycle or queue activity
type Event struct {
	Message    string            `json:"message,omitempty"`
	Connection ConnectionDetails `json:"connection"`
	Source     SourceDetails     `json:"source"`
	Instance   InstanceDetails   `json:"instance"`
	Queue      *QueueDetails     `json:"queue,omitempty"`
	Error      *ErrorDetails     `json:"error,omitempty"` // Only present for error events
}

// CreateEventParams defines parameters for creating a new log event
type CreateEventParams struct {
	Message      string
	ConnectionID *id.ULID
	ClientID     *id.ULID
	Alias        *string
	ChannelName  *string
	APIKey       *string
	RemoteAddr   string
	UserAgent    *string
	Origin       *string
	Site         string
	Error        *ErrorDetails
}

// CreateEvent creates a new log event instance with structured data
func CreateEvent(params CreateEventParams) *Event {
	return &Event{
		Message: params.Message,
		Connection: ConnectionDetails{
			ConnectionID: params.ConnectionID,
			ClientID:     params.ClientID,
			Alias:        params.Alias,
			ChannelName:  params.ChannelName,
			APIKey:       params.APIKey,
		},
		Source: SourceDetails{
			RemoteAddr: params.RemoteAddr,
			UserAgent:  params.UserAgent,
			Origin:     params.Origin,
		},
		Instance: InstanceDetails{
			Site: params.Site,
		},
		Error: params.Error,
	}
}

// CreateQueueEventParams defines parameters for creating a queue activity log event
type CreateQueueEventParams struct {
	Message    string
	QueueName  string
	JobID      *id.ULID
	Status     string
	WorkerID   string
	WorkerName string
	Attempt    int
	Site       string
	Error      *ErrorDetails
}

// CreateQueueEvent creates a log event for queue activity
func CreateQueueEvent(params CreateQueueEventParams) *Event {
	return &Event{
		Message: params.Message,
		Instance: InstanceDetails{
			Site: params.Site,
		},
		Queue: &QueueDetails{
			QueueName:  params.QueueName,
			JobID:      params.JobID,
			Status:     params.Status,
			WorkerID:   params.WorkerID,
			WorkerName: params.WorkerName,
			Attempt:    params.Attempt,
		},
		Error: params.Error,
	}
}
