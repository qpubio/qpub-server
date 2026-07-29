package envelope

import (
	"time"

	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Direction is server-centric message flow.
type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

// Source identifies where an envelope entered or will leave the runtime.
type Source string

const (
	SourceWebSocket Source = "websocket"
	SourceREST      Source = "rest"
	SourceNATS      Source = "nats"
	SourceInternal  Source = "internal"
)

// Envelope is the internal unit routed through the messaging runtime.
// Payload holds the marshaled wire message (protocol.DataMessage bytes).
type Envelope struct {
	id            id.ULID
	direction     Direction
	projectID     id.Int
	channel       string
	payload       []byte
	source        Source
	connectionID  id.ULID
	clientID      id.ULID
	publishedAt   time.Time
	skipTelemetry bool
}

// CreateParams holds parameters for constructing an envelope.
type CreateParams struct {
	Direction     Direction
	ProjectID     id.Int
	Channel       string
	Payload       []byte
	Source        Source
	ConnectionID  id.ULID
	ClientID      id.ULID
	PublishedAt   time.Time
	SkipTelemetry bool
}

// New creates a new envelope with a generated ID.
func New(params CreateParams) *Envelope {
	envID := id.NewULID()
	publishedAt := params.PublishedAt
	if publishedAt.IsZero() {
		publishedAt = clock.Now()
	}

	payload := params.Payload
	if payload != nil {
		payload = append([]byte(nil), payload...)
	}

	return &Envelope{
		id:            envID,
		direction:     params.Direction,
		projectID:     params.ProjectID,
		channel:       params.Channel,
		payload:       payload,
		source:        params.Source,
		connectionID:  params.ConnectionID,
		clientID:      params.ClientID,
		publishedAt:   publishedAt,
		skipTelemetry: params.SkipTelemetry,
	}
}

// WithID returns a copy of the envelope with the given ID (e.g. reuse publish envelope id).
func (e *Envelope) WithID(envID id.ULID) *Envelope {
	if e == nil {
		return nil
	}
	copy := *e
	copy.id = envID
	return &copy
}

func (e *Envelope) ID() id.ULID           { return e.id }
func (e *Envelope) Direction() Direction  { return e.direction }
func (e *Envelope) ProjectID() id.Int     { return e.projectID }
func (e *Envelope) Channel() string       { return e.channel }
func (e *Envelope) Payload() []byte       { return e.payload }
func (e *Envelope) Source() Source        { return e.source }
func (e *Envelope) ConnectionID() id.ULID { return e.connectionID }
func (e *Envelope) ClientID() id.ULID     { return e.clientID }
func (e *Envelope) PublishedAt() time.Time {
	return e.publishedAt
}
func (e *Envelope) SkipTelemetry() bool { return e.skipTelemetry }

// Size returns the payload size in bytes.
func (e *Envelope) Size() int64 {
	if e == nil {
		return 0
	}
	return int64(len(e.payload))
}

// IsInbound reports whether the envelope represents traffic entering the platform.
func (e *Envelope) IsInbound() bool {
	return e != nil && e.direction == DirectionInbound
}

// IsOutbound reports whether the envelope represents traffic leaving to subscribers.
func (e *Envelope) IsOutbound() bool {
	return e != nil && e.direction == DirectionOutbound
}
