package router

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	msgTelemetry "github.com/qpubio/qpub-server/internal/domain/messaging/telemetry"
	projectLog "github.com/qpubio/qpub-server/internal/domain/project/log"
	logBroadcast "github.com/qpubio/qpub-server/internal/domain/project/log/broadcast"
	domainBackpressure "github.com/qpubio/qpub-server/internal/domain/queue/backpressure"
	domainBroker "github.com/qpubio/qpub-server/internal/domain/queue/broker"
	"github.com/qpubio/qpub-server/internal/domain/queue/envelope"
	"github.com/qpubio/qpub-server/internal/domain/queue/execution"
	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
	"github.com/qpubio/qpub-server/internal/domain/queue/receipt"
	domainRouter "github.com/qpubio/qpub-server/internal/domain/queue/router"
	queueTelemetry "github.com/qpubio/qpub-server/internal/domain/queue/telemetry"
	domainWorker "github.com/qpubio/qpub-server/internal/domain/queue/worker"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"gorm.io/gorm"
)

type brokerMessage struct {
	JobID    string          `json:"jobId"`
	Payload  json.RawMessage `json:"payload"`
	Attempt  int             `json:"attempt"`
	Sequence uint64          `json:"sequence"`
}

type Service struct {
	jobRepo        domainJob.Repository
	queueRepo      domainQueue.Repository
	broker         domainBroker.Repository
	gatekeeper     domainBackpressure.Gatekeeper
	telemetry      queueTelemetry.Service
	msgTelemetry   msgTelemetry.Service
	logBroadcaster logBroadcast.Service
	workerSvc      domainWorker.Service
	instanceID     id.ULID
	logger         logger.Service
	queueService   domainQueue.Service
}

func NewService(
	jobRepo domainJob.Repository,
	queueRepo domainQueue.Repository,
	broker domainBroker.Repository,
	gatekeeper domainBackpressure.Gatekeeper,
	telemetry queueTelemetry.Service,
	msgTelemetryService msgTelemetry.Service,
	logBroadcaster logBroadcast.Service,
	workerSvc domainWorker.Service,
	instanceID id.ULID,
	queueService domainQueue.Service,
	logger logger.Service,
) domainRouter.Service {
	return &Service{
		jobRepo:        jobRepo,
		queueRepo:      queueRepo,
		broker:         broker,
		gatekeeper:     gatekeeper,
		telemetry:      telemetry,
		msgTelemetry:   msgTelemetryService,
		logBroadcaster: logBroadcaster,
		workerSvc:      workerSvc,
		instanceID:     instanceID,
		queueService:   queueService,
		logger:         logger,
	}
}

func (s *Service) Enqueue(ctx context.Context, req domainJob.EnqueueRequest) (*receipt.Receipt, *domainJob.Job, error) {
	allowed, err := s.gatekeeper.AllowEnqueue(req.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	if !allowed {
		env := envelope.New(envelope.CreateParams{
			Direction: envelope.DirectionInbound,
			ProjectID: req.ProjectID,
			QueueName: req.QueueName,
			Source:    envelope.SourceREST,
		})
		return receipt.IngressNack(env.ID(), env.JobID(), "rate limit exceeded"), nil, fmt.Errorf("enqueue rate limit exceeded")
	}

	q, err := s.queueService.Ensure(domainQueue.CreateParams{
		ProjectID:        req.ProjectID,
		Name:             req.QueueName,
		ExecutionProfile: execution.ProfileExternal,
	})
	if err != nil {
		return nil, nil, err
	}

	if err := execution.ValidateProfile(q.ExecutionProfile); err != nil {
		return nil, nil, err
	}

	if req.IdempotencyKey != "" {
		existing, err := s.jobRepo.FindByIdempotencyKey(req.ProjectID, req.QueueName, req.IdempotencyKey)
		if err == nil {
			env := envelope.New(envelope.CreateParams{
				Direction: envelope.DirectionInbound,
				ProjectID: req.ProjectID,
				QueueName: req.QueueName,
				JobID:     existing.ID,
				Source:    envelope.SourceREST,
			})
			return receipt.IngressAck(env.ID(), existing.ID), existing, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, nil, err
		}
	}

	scheduleAt := req.ScheduleAt
	if req.Delay > 0 {
		t := clock.Now().Add(req.Delay)
		scheduleAt = &t
	}

	jobEntity, err := domainJob.Enqueue(domainJob.CreateParams{
		ProjectID:      req.ProjectID,
		QueueName:      req.QueueName,
		Payload:        req.Payload,
		IdempotencyKey: req.IdempotencyKey,
		MaxAttempts:    q.MaxAttempts,
		ScheduleAt:     scheduleAt,
		Metadata:       req.Metadata,
	})
	if err != nil {
		return nil, nil, err
	}

	if err := s.jobRepo.Create(jobEntity); err != nil {
		return nil, nil, err
	}

	env := envelope.New(envelope.CreateParams{
		Direction: envelope.DirectionInbound,
		ProjectID: req.ProjectID,
		QueueName: req.QueueName,
		JobID:     jobEntity.ID,
		Payload:   req.Payload,
		Source:    envelope.SourceREST,
	})

	if jobEntity.IsClaimable(clock.Now()) {
		if err := s.publishToBroker(ctx, q, jobEntity); err != nil {
			s.logger.Error(log.Queue, "Failed to publish job to broker: %v", err)
		}
	}

	s.telemetry.RecordEnqueue(req.ProjectID, env.Size())
	s.recordMessagingInbound(req.ProjectID, req.QueueName, env.ID(), int64(len(req.Payload)))
	s.publishQueueLog(req.ProjectID, projectLog.EventQueueJobEnqueued, projectLog.CreateQueueEventParams{
		Message:   "Job enqueued",
		QueueName: req.QueueName,
		JobID:     &jobEntity.ID,
		Status:    string(jobEntity.Status),
		Attempt:   jobEntity.Attempt,
	})
	return receipt.IngressAck(env.ID(), jobEntity.ID), jobEntity, nil
}

func (s *Service) Dequeue(ctx context.Context, req domainJob.DequeueRequest) ([]domainJob.Job, error) {
	allowed, err := s.gatekeeper.AllowDequeue(req.ProjectID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, fmt.Errorf("dequeue rate limit exceeded")
	}

	batchSize := req.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}

	attemptClaim := func() ([]domainJob.Job, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		claimed, claimErr := s.jobRepo.ClaimPending(req.ProjectID, req.QueueName, req.WorkerID, batchSize, clock.Now())
		if claimErr != nil {
			return nil, claimErr
		}
		if len(claimed) > 0 {
			s.touchRegisteredWorker(req.ProjectID, req.WorkerID)
		}
		for i := range claimed {
			j := claimed[i]
			s.recordMessagingOutbound(req.ProjectID, req.QueueName, j.ID, int64(len(j.Payload)))
			jobID := j.ID
			s.publishQueueLog(req.ProjectID, projectLog.EventQueueJobClaimed, projectLog.CreateQueueEventParams{
				Message:   "Job claimed",
				QueueName: req.QueueName,
				JobID:     &jobID,
				Status:    string(j.Status),
				WorkerID:  req.WorkerID,
				Attempt:   j.Attempt,
			})
		}
		return claimed, nil
	}

	// Wait <= 0: single attempt, no long-poll.
	if req.Wait <= 0 {
		return attemptClaim()
	}

	deadline := clock.Now().Add(req.Wait)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if !clock.Now().Before(deadline) {
			return []domainJob.Job{}, nil
		}

		claimed, claimErr := attemptClaim()
		if claimErr != nil {
			return nil, claimErr
		}
		if len(claimed) > 0 {
			return claimed, nil
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return []domainJob.Job{}, nil
		}
		sleep := 500 * time.Millisecond
		if remaining < sleep {
			sleep = remaining
		}
		timer := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (s *Service) Ack(ctx context.Context, req domainJob.AckRequest) (*receipt.Receipt, error) {
	j, err := s.jobRepo.FindByID(req.ProjectID, req.QueueName, req.JobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainJob.ErrNotFound
		}
		return nil, err
	}

	if j.Status != domainJob.StatusRunning {
		return nil, domainJob.ErrInvalidTransition
	}
	if err := assertJobWorker(j, req.WorkerID); err != nil {
		return nil, err
	}

	j.MarkCompleted(req.Result)
	if err := s.jobRepo.Update(j); err != nil {
		return nil, err
	}

	if j.StartedAt != nil {
		duration := clock.Now().Sub(*j.StartedAt).Milliseconds()
		s.telemetry.RecordCompleted(req.ProjectID, duration)
	}

	env := envelope.New(envelope.CreateParams{
		Direction: envelope.DirectionOutbound,
		ProjectID: req.ProjectID,
		QueueName: req.QueueName,
		JobID:     j.ID,
		Source:    envelope.SourceInternal,
	})

	s.publishQueueLog(req.ProjectID, projectLog.EventQueueJobCompleted, projectLog.CreateQueueEventParams{
		Message:   "Job completed",
		QueueName: req.QueueName,
		JobID:     &j.ID,
		Status:    string(j.Status),
		WorkerID:  req.WorkerID,
		Attempt:   j.Attempt,
	})

	return receipt.New(receipt.CreateParams{
		EnvelopeID: env.ID(),
		JobID:      j.ID,
		Kind:       receipt.KindAck,
		Stage:      receipt.StageWorker,
		Outcome:    receipt.OutcomeCompleted,
	}), nil
}

func (s *Service) Nack(ctx context.Context, req domainJob.NackRequest) (*receipt.Receipt, error) {
	j, err := s.jobRepo.FindByID(req.ProjectID, req.QueueName, req.JobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainJob.ErrNotFound
		}
		return nil, err
	}

	if j.Status != domainJob.StatusRunning {
		return nil, domainJob.ErrInvalidTransition
	}
	if err := assertJobWorker(j, req.WorkerID); err != nil {
		return nil, err
	}

	env := envelope.New(envelope.CreateParams{
		Direction: envelope.DirectionOutbound,
		ProjectID: req.ProjectID,
		QueueName: req.QueueName,
		JobID:     j.ID,
		Source:    envelope.SourceInternal,
		Attempt:   j.Attempt,
	})

	if j.CanRetry() {
		delay := req.RetryDelay
		if delay <= 0 {
			delay = time.Duration(j.Attempt*j.Attempt) * time.Minute
		}
		j.MarkRetry(delay)
		if err := s.jobRepo.Update(j); err != nil {
			return nil, err
		}
		s.telemetry.RecordRetried(req.ProjectID)
		s.publishQueueLog(req.ProjectID, projectLog.EventQueueJobNacked, projectLog.CreateQueueEventParams{
			Message:   "Job nacked",
			QueueName: req.QueueName,
			JobID:     &j.ID,
			Status:    string(j.Status),
			WorkerID:  req.WorkerID,
			Attempt:   j.Attempt,
		})
		s.publishQueueLog(req.ProjectID, projectLog.EventQueueJobRetried, projectLog.CreateQueueEventParams{
			Message:   "Job scheduled for retry",
			QueueName: req.QueueName,
			JobID:     &j.ID,
			Status:    string(j.Status),
			WorkerID:  req.WorkerID,
			Attempt:   j.Attempt,
		})

		q, qErr := s.queueRepo.FindByProjectAndName(req.ProjectID, req.QueueName)
		if qErr == nil && j.IsClaimable(clock.Now()) {
			_ = s.publishToBroker(ctx, *q, j)
		}

		return receipt.New(receipt.CreateParams{
			EnvelopeID: env.ID(),
			JobID:      j.ID,
			Kind:       receipt.KindNack,
			Stage:      receipt.StageWorker,
			Outcome:    receipt.OutcomeRetried,
			Reason:     req.Reason,
		}), nil
	}

	j.MarkDLQ(req.Reason)
	if err := s.jobRepo.Update(j); err != nil {
		return nil, err
	}
	s.telemetry.RecordDLQ(req.ProjectID)
	s.publishQueueLog(req.ProjectID, projectLog.EventQueueJobNacked, projectLog.CreateQueueEventParams{
		Message:   "Job nacked",
		QueueName: req.QueueName,
		JobID:     &j.ID,
		Status:    string(j.Status),
		WorkerID:  req.WorkerID,
		Attempt:   j.Attempt,
	})
	s.publishQueueLog(req.ProjectID, projectLog.EventQueueJobDLQ, projectLog.CreateQueueEventParams{
		Message:   "Job moved to DLQ",
		QueueName: req.QueueName,
		JobID:     &j.ID,
		Status:    string(j.Status),
		WorkerID:  req.WorkerID,
		Attempt:   j.Attempt,
	})

	qName := domainQueue.NewName(req.QueueName, req.ProjectID)
	_ = s.broker.PublishDLQ(ctx, qName.DLQSubject(), j.Payload)

	return receipt.New(receipt.CreateParams{
		EnvelopeID: env.ID(),
		JobID:      j.ID,
		Kind:       receipt.KindNack,
		Stage:      receipt.StageWorker,
		Outcome:    receipt.OutcomeFailed,
		Reason:     req.Reason,
	}), nil
}

func (s *Service) Cancel(ctx context.Context, projectID id.Int, queueName string, jobID id.ULID) (*receipt.Receipt, *domainJob.Job, error) {
	j, err := s.jobRepo.FindByID(projectID, queueName, jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, domainJob.ErrNotFound
		}
		return nil, nil, err
	}

	switch j.Status {
	case domainJob.StatusCompleted, domainJob.StatusCancelled, domainJob.StatusDLQ:
		return nil, nil, domainJob.ErrInvalidTransition
	}

	j.MarkCancelled()
	if err := s.jobRepo.Update(j); err != nil {
		return nil, nil, err
	}

	env := envelope.New(envelope.CreateParams{
		Direction: envelope.DirectionInbound,
		ProjectID: projectID,
		QueueName: queueName,
		JobID:     j.ID,
		Source:    envelope.SourceREST,
	})

	s.publishQueueLog(projectID, projectLog.EventQueueJobCancelled, projectLog.CreateQueueEventParams{
		Message:   "Job cancelled",
		QueueName: queueName,
		JobID:     &j.ID,
		Status:    string(j.Status),
		Attempt:   j.Attempt,
	})

	return receipt.New(receipt.CreateParams{
		EnvelopeID: env.ID(),
		JobID:      j.ID,
		Kind:       receipt.KindAck,
		Stage:      StageIngressCancel(),
		Outcome:    receipt.OutcomeCancelled,
	}), j, nil
}

func StageIngressCancel() receipt.Stage {
	return receipt.StageIngress
}

func (s *Service) EnsureQueueListening(projectID id.Int, queueName string) error {
	q, err := s.queueService.Ensure(domainQueue.CreateParams{
		ProjectID:        projectID,
		Name:             queueName,
		ExecutionProfile: execution.ProfileExternal,
	})
	if err != nil {
		return err
	}

	name := q.QueueName()
	return s.broker.EnsureStream(name.Subject(), q.Retention)
}

func (s *Service) publishToBroker(ctx context.Context, q domainQueue.Queue, j *domainJob.Job) error {
	name := q.QueueName()
	msg := brokerMessage{
		JobID:   string(j.ID),
		Payload: j.Payload,
		Attempt: j.Attempt,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	seq, err := s.broker.Publish(ctx, name.Subject(), data)
	if err != nil {
		return err
	}

	j.BrokerSequence = seq
	return s.jobRepo.Update(j)
}

func (s *Service) recordMessagingInbound(projectID id.Int, queueName string, envelopeID id.ULID, byteSize int64) {
	if s.msgTelemetry == nil {
		return
	}
	evt := msgTelemetry.NewEvent(
		msgTelemetry.TypeInboundAccepted,
		projectID,
		s.instanceID,
		msgTelemetry.InboundAcceptedData{
			EnvelopeID: envelopeID,
			Channel:    "queue:" + queueName,
			ByteSize:   byteSize,
			Source:     "queue",
		},
	)
	if err := s.msgTelemetry.Record(evt); err != nil {
		s.logger.Warn(log.Queue, "Failed to record queue inbound telemetry: %v", err)
	}
}

func (s *Service) recordMessagingOutbound(projectID id.Int, queueName string, jobID id.ULID, byteSize int64) {
	if s.msgTelemetry == nil {
		return
	}
	evt := msgTelemetry.NewEvent(
		msgTelemetry.TypeOutboundDelivered,
		projectID,
		s.instanceID,
		msgTelemetry.OutboundDeliveredData{
			EnvelopeID: jobID,
			Channel:    "queue:" + queueName,
			ByteSize:   byteSize,
		},
	)
	if err := s.msgTelemetry.Record(evt); err != nil {
		s.logger.Warn(log.Queue, "Failed to record queue outbound telemetry: %v", err)
	}
}

func (s *Service) publishQueueLog(projectID id.Int, eventType projectLog.EventType, params projectLog.CreateQueueEventParams) {
	if s.logBroadcaster == nil {
		return
	}
	event := projectLog.CreateQueueEvent(params)
	if err := s.logBroadcaster.PublishLog(projectID, eventType, *event); err != nil {
		s.logger.Warn(log.Queue, "Failed to publish queue log event: %v", err)
	}
}

// PublishDueScheduled promotes scheduled jobs whose time has arrived.
func (s *Service) PublishDueScheduled(ctx context.Context) error {
	jobs, err := s.jobRepo.FindDueScheduled(100, clock.Now())
	if err != nil {
		return err
	}

	for i := range jobs {
		j := &jobs[i]
		if !j.IsClaimable(clock.Now()) {
			continue
		}
		q, err := s.queueRepo.FindByProjectAndName(j.ProjectID, j.QueueName)
		if err != nil {
			continue
		}
		if j.Status == domainJob.StatusScheduled {
			j.MarkPendingForSchedule()
			if err := s.jobRepo.Update(j); err != nil {
				s.logger.Warn(log.Queue, "Failed to promote scheduled job job=%s err=%v", j.ID, err)
				continue
			}
		}
		if err := s.publishToBroker(ctx, *q, j); err != nil {
			s.logger.Warn(log.Queue, "Failed to publish due job to broker job=%s err=%v", j.ID, err)
		}
	}
	return nil
}

const reclaimBatchSize = 500

// ReclaimExpired returns running jobs whose visibility lease expired to pending.
func (s *Service) ReclaimExpired(ctx context.Context, limit int) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	if limit <= 0 {
		limit = reclaimBatchSize
	}
	n, err := s.jobRepo.ReclaimExpired(clock.Now(), limit, domainQueue.DefaultVisibilityTimeout)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		s.logger.Info(log.Queue, "Reclaimed expired running jobs count=%d", n)
	}
	return n, nil
}

// touchRegisteredWorker bumps last_seen_at when worker_id matches a registered row.
func (s *Service) touchRegisteredWorker(projectID id.Int, workerID string) {
	if s.workerSvc == nil || workerID == "" {
		return
	}
	if _, err := s.workerSvc.Heartbeat(projectID, id.ULID(workerID)); err != nil {
		// Unregistered / anonymous worker_ids are expected; ignore.
		return
	}
}

func assertJobWorker(j *domainJob.Job, workerID string) error {
	if j.WorkerID == "" {
		// Legacy rows without a claimer remain completable.
		return nil
	}
	if workerID != j.WorkerID {
		return domainJob.ErrWorkerMismatch
	}
	return nil
}
