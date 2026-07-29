package envelope

import (
	"time"

	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Direction is server-centric job flow.
type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

// Source identifies where a job envelope entered the runtime.
type Source string

const (
	SourceREST      Source = "rest"
	SourceWebSocket Source = "websocket"
	SourceScheduler Source = "scheduler"
	SourceInternal  Source = "internal"
	SourceWebhook   Source = "webhook"
)

// Envelope is the internal unit routed through the queue runtime.
type Envelope struct {
	id           id.ULID
	direction    Direction
	projectID    id.Int
	queueName    string
	jobID        id.ULID
	payload      []byte
	source       Source
	attempt      int
	enqueuedAt   time.Time
	skipTelemetry bool
}

type CreateParams struct {
	Direction     Direction
	ProjectID     id.Int
	QueueName     string
	JobID         id.ULID
	Payload       []byte
	Source        Source
	Attempt       int
	EnqueuedAt    time.Time
	SkipTelemetry bool
}

func New(params CreateParams) *Envelope {
	enqueuedAt := params.EnqueuedAt
	if enqueuedAt.IsZero() {
		enqueuedAt = clock.Now()
	}

	payload := params.Payload
	if payload != nil {
		payload = append([]byte(nil), payload...)
	}

	jobID := params.JobID
	if jobID == "" {
		jobID = id.NewULID()
	}

	return &Envelope{
		id:            id.NewULID(),
		direction:     params.Direction,
		projectID:     params.ProjectID,
		queueName:     params.QueueName,
		jobID:         jobID,
		payload:       payload,
		source:        params.Source,
		attempt:       params.Attempt,
		enqueuedAt:    enqueuedAt,
		skipTelemetry: params.SkipTelemetry,
	}
}

func (e *Envelope) ID() id.ULID           { return e.id }
func (e *Envelope) Direction() Direction  { return e.direction }
func (e *Envelope) ProjectID() id.Int     { return e.projectID }
func (e *Envelope) QueueName() string     { return e.queueName }
func (e *Envelope) JobID() id.ULID        { return e.jobID }
func (e *Envelope) Payload() []byte       { return e.payload }
func (e *Envelope) Source() Source        { return e.source }
func (e *Envelope) Attempt() int          { return e.attempt }
func (e *Envelope) EnqueuedAt() time.Time { return e.enqueuedAt }
func (e *Envelope) SkipTelemetry() bool   { return e.skipTelemetry }

func (e *Envelope) Size() int64 {
	if e == nil {
		return 0
	}
	return int64(len(e.payload))
}
