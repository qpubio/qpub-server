package receipt

import (
	"time"

	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

type Kind string

const (
	KindAck  Kind = "ack"
	KindNack Kind = "nack"
)

type Stage string

const (
	StageIngress Stage = "ingress"
	StageWorker  Stage = "worker"
)

type Outcome string

const (
	OutcomeAccepted  Outcome = "accepted"
	OutcomeRejected  Outcome = "rejected"
	OutcomeQueued    Outcome = "queued"
	OutcomeClaimed   Outcome = "claimed"
	OutcomeCompleted Outcome = "completed"
	OutcomeRetried   Outcome = "retried"
	OutcomeFailed    Outcome = "failed"
	OutcomeCancelled Outcome = "cancelled"
)

type Receipt struct {
	id         id.ULID
	envelopeID id.ULID
	jobID      id.ULID
	kind       Kind
	stage      Stage
	outcome    Outcome
	reason     string
	recordedAt time.Time
}

type CreateParams struct {
	EnvelopeID id.ULID
	JobID      id.ULID
	Kind       Kind
	Stage      Stage
	Outcome    Outcome
	Reason     string
	RecordedAt time.Time
}

func New(params CreateParams) *Receipt {
	recordedAt := params.RecordedAt
	if recordedAt.IsZero() {
		recordedAt = clock.Now()
	}

	return &Receipt{
		id:         id.NewULID(),
		envelopeID: params.EnvelopeID,
		jobID:      params.JobID,
		kind:       params.Kind,
		stage:      params.Stage,
		outcome:    params.Outcome,
		reason:     params.Reason,
		recordedAt: recordedAt,
	}
}

func (r *Receipt) ID() id.ULID           { return r.id }
func (r *Receipt) EnvelopeID() id.ULID   { return r.envelopeID }
func (r *Receipt) JobID() id.ULID        { return r.jobID }
func (r *Receipt) Kind() Kind            { return r.kind }
func (r *Receipt) Stage() Stage          { return r.stage }
func (r *Receipt) Outcome() Outcome      { return r.outcome }
func (r *Receipt) Reason() string        { return r.reason }
func (r *Receipt) RecordedAt() time.Time { return r.recordedAt }

func IngressAck(envelopeID, jobID id.ULID) *Receipt {
	return New(CreateParams{
		EnvelopeID: envelopeID,
		JobID:      jobID,
		Kind:       KindAck,
		Stage:      StageIngress,
		Outcome:    OutcomeAccepted,
	})
}

func IngressNack(envelopeID, jobID id.ULID, reason string) *Receipt {
	return New(CreateParams{
		EnvelopeID: envelopeID,
		JobID:      jobID,
		Kind:       KindNack,
		Stage:      StageIngress,
		Outcome:    OutcomeRejected,
		Reason:     reason,
	})
}
