package receipt

import (
	"time"

	"github.com/qpubio/qpub-server/internal/domain/messaging/protocol"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Kind distinguishes positive and negative receipts.
type Kind string

const (
	KindAck  Kind = "ack"
	KindNack Kind = "nack"
)

// Stage identifies where in the pipeline the receipt was produced.
type Stage string

const (
	StageIngress Stage = "ingress"
	StageEgress  Stage = "egress"
)

// Outcome describes the result of a delivery attempt.
type Outcome string

const (
	OutcomeAccepted  Outcome = "accepted"
	OutcomeRejected  Outcome = "rejected"
	OutcomeQueued    Outcome = "queued"
	OutcomeDelivered Outcome = "delivered"
	OutcomeDropped   Outcome = "dropped"
	OutcomeFailed    Outcome = "failed"
)

// Receipt records the result of processing an envelope at a pipeline stage.
type Receipt struct {
	id         id.ULID
	envelopeID id.ULID
	kind       Kind
	stage      Stage
	outcome    Outcome
	reason     string
	errorCode  protocol.Code
	recordedAt time.Time
}

// CreateParams holds parameters for constructing a receipt.
type CreateParams struct {
	EnvelopeID id.ULID
	Kind       Kind
	Stage      Stage
	Outcome    Outcome
	Reason     string
	ErrorCode  protocol.Code
	RecordedAt time.Time
}

// New creates a receipt for an envelope processing result.
func New(params CreateParams) *Receipt {
	recordedAt := params.RecordedAt
	if recordedAt.IsZero() {
		recordedAt = clock.Now()
	}

	return &Receipt{
		id:         id.NewULID(),
		envelopeID: params.EnvelopeID,
		kind:       params.Kind,
		stage:      params.Stage,
		outcome:    params.Outcome,
		reason:     params.Reason,
		errorCode:  params.ErrorCode,
		recordedAt: recordedAt,
	}
}

func (r *Receipt) ID() id.ULID              { return r.id }
func (r *Receipt) EnvelopeID() id.ULID      { return r.envelopeID }
func (r *Receipt) Kind() Kind               { return r.kind }
func (r *Receipt) Stage() Stage             { return r.stage }
func (r *Receipt) Outcome() Outcome         { return r.outcome }
func (r *Receipt) Reason() string           { return r.reason }
func (r *Receipt) ErrorCode() protocol.Code { return r.errorCode }
func (r *Receipt) RecordedAt() time.Time    { return r.recordedAt }

// IsAck reports whether the receipt is an acknowledgment.
func (r *Receipt) IsAck() bool {
	return r != nil && r.kind == KindAck
}

// IsNack reports whether the receipt is a negative acknowledgment.
func (r *Receipt) IsNack() bool {
	return r != nil && r.kind == KindNack
}

// IsSuccess reports whether the outcome represents successful processing for the stage.
func (r *Receipt) IsSuccess() bool {
	if r == nil {
		return false
	}
	switch r.outcome {
	case OutcomeAccepted, OutcomeQueued, OutcomeDelivered:
		return true
	default:
		return false
	}
}

// IngressAck builds an ingress acknowledgment for an accepted publish.
func IngressAck(envelopeID id.ULID) *Receipt {
	return New(CreateParams{
		EnvelopeID: envelopeID,
		Kind:       KindAck,
		Stage:      StageIngress,
		Outcome:    OutcomeAccepted,
	})
}

// IngressNack builds an ingress negative acknowledgment.
func IngressNack(envelopeID id.ULID, reason string, code protocol.Code) *Receipt {
	return New(CreateParams{
		EnvelopeID: envelopeID,
		Kind:       KindNack,
		Stage:      StageIngress,
		Outcome:    OutcomeRejected,
		Reason:     reason,
		ErrorCode:  code,
	})
}

// EgressDelivered builds an egress receipt for a successful socket write.
func EgressDelivered(envelopeID id.ULID) *Receipt {
	return New(CreateParams{
		EnvelopeID: envelopeID,
		Kind:       KindAck,
		Stage:      StageEgress,
		Outcome:    OutcomeDelivered,
	})
}

// EgressDropped builds an egress receipt for a backpressure drop.
func EgressDropped(envelopeID id.ULID, reason string) *Receipt {
	return New(CreateParams{
		EnvelopeID: envelopeID,
		Kind:       KindNack,
		Stage:      StageEgress,
		Outcome:    OutcomeDropped,
		Reason:     reason,
	})
}
