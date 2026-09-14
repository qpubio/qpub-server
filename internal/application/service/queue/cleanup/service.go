package cleanup

import (
	"context"
	"errors"
	"fmt"

	apiKeyDomain "github.com/qpubio/qpub-server/internal/domain/apikey"
	domainBroker "github.com/qpubio/qpub-server/internal/domain/queue/broker"
	domainCleanup "github.com/qpubio/qpub-server/internal/domain/queue/cleanup"
	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
	domainWorker "github.com/qpubio/qpub-server/internal/domain/queue/worker"
	"github.com/qpubio/qpub-server/internal/domain/tenant"
	"github.com/qpubio/qpub-server/internal/config/infrastructure"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"gorm.io/gorm"
)

const defaultBatchSize = 1000

type Service struct {
	jobRepo       domainJob.Repository
	queueRepo     domainQueue.Repository
	workerRepo    domainWorker.Repository
	tenantRepo    tenant.Repository
	broker        domainBroker.Repository
	apiKeyService apiKeyDomain.Service
	cfg           infrastructure.Queue
	logger        logger.Service
}

func NewService(
	jobRepo domainJob.Repository,
	queueRepo domainQueue.Repository,
	workerRepo domainWorker.Repository,
	tenantRepo tenant.Repository,
	broker domainBroker.Repository,
	apiKeyService apiKeyDomain.Service,
	cfg infrastructure.Queue,
	logger logger.Service,
) domainCleanup.Service {
	return &Service{
		jobRepo:       jobRepo,
		queueRepo:     queueRepo,
		workerRepo:    workerRepo,
		tenantRepo:    tenantRepo,
		broker:        broker,
		apiKeyService: apiKeyService,
		cfg:           cfg,
		logger:        logger,
	}
}

func (s *Service) batchSize() int {
	if s.cfg.Cleanup.BatchSize > 0 {
		return s.cfg.Cleanup.BatchSize
	}
	return defaultBatchSize
}

func (s *Service) RequestDeleteQueue(ctx context.Context, projectID id.Int, queueName string, force bool) (bool, error) {
	q, err := s.queueRepo.FindByProjectAndName(projectID, queueName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, domainQueue.ErrNotFound
		}
		return false, err
	}
	if q.Status == domainQueue.StatusDeleting {
		return false, nil
	}

	active, err := s.jobRepo.CountActiveByQueue(projectID, queueName)
	if err != nil {
		return false, err
	}
	if active > 0 && !force {
		return false, domainQueue.ErrActiveJobs
	}

	now := clock.Now()
	q.Status = domainQueue.StatusDeleting
	q.UpdatedAt = now
	if err := s.queueRepo.Update(q); err != nil {
		return false, err
	}

	if force && active > 0 {
		if _, err := s.jobRepo.ForceCancelActiveByQueue(projectID, queueName, now); err != nil {
			return false, err
		}
	}

	total, err := s.jobRepo.CountByQueue(projectID, queueName)
	if err != nil {
		return false, err
	}
	if total == 0 {
		if err := s.cascadeQueue(ctx, projectID, queueName); err != nil {
			return false, err
		}
		return true, nil
	}

	go func() {
		if err := s.cascadeQueue(context.Background(), projectID, queueName); err != nil {
			s.logger.Error(log.Queue, "Queue cascade failed project=%d queue=%s err=%v", projectID, queueName, err)
		}
	}()
	return false, nil
}

func (s *Service) RequestDeleteTenant(ctx context.Context, tenantID id.Int) error {
	t, err := s.tenantRepo.FindTenant(tenantID)
	if err != nil {
		return err
	}
	if t == nil {
		return fmt.Errorf("tenant not found")
	}
	if t.Status == tenant.StatusDeleting {
		return nil
	}

	now := clock.Now()
	t.Status = tenant.StatusDeleting
	t.UpdatedAt = now
	if err := s.tenantRepo.UpdateTenant(*t); err != nil {
		return err
	}

	go func() {
		if err := s.cascadeTenant(context.Background(), tenantID); err != nil {
			s.logger.Error(log.Queue, "Tenant cascade failed tenant=%d err=%v", tenantID, err)
		}
	}()
	return nil
}

func (s *Service) ResumePendingCascades(ctx context.Context) error {
	queues, err := s.queueRepo.ListDeleting(s.batchSize())
	if err != nil {
		return err
	}
	for _, q := range queues {
		q := q
		if err := s.cascadeQueue(ctx, q.ProjectID, q.Name); err != nil {
			s.logger.Warn(log.Queue, "Resume queue cascade failed project=%d queue=%s err=%v", q.ProjectID, q.Name, err)
		}
	}

	tenants, err := s.tenantRepo.ListDeleting(s.batchSize())
	if err != nil {
		return err
	}
	for _, t := range tenants {
		if err := s.cascadeTenant(ctx, t.ID); err != nil {
			s.logger.Warn(log.Queue, "Resume tenant cascade failed tenant=%d err=%v", t.ID, err)
		}
	}
	return nil
}

func (s *Service) PurgeTerminalJobs(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	now := clock.Now()
	successCutoff := now.Add(-s.cfg.Cleanup.JobSuccessRetention)
	failureCutoff := now.Add(-s.cfg.Cleanup.JobFailureRetention)

	for {
		n, err := s.jobRepo.PurgeTerminalBefore(
			[]domainJob.Status{domainJob.StatusCompleted, domainJob.StatusCancelled},
			successCutoff,
			s.batchSize(),
		)
		if err != nil {
			return err
		}
		if n == 0 {
			break
		}
	}

	for {
		n, err := s.jobRepo.PurgeTerminalBefore(
			[]domainJob.Status{domainJob.StatusFailed, domainJob.StatusDLQ},
			failureCutoff,
			s.batchSize(),
		)
		if err != nil {
			return err
		}
		if n == 0 {
			break
		}
	}
	return nil
}

func (s *Service) PurgeStaleWorkers(ctx context.Context) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	cutoff := clock.Now().Add(-s.cfg.Cleanup.WorkerStaleDelete)
	workers, err := s.workerRepo.ListStale(cutoff, s.batchSize())
	if err != nil {
		return 0, err
	}

	removed := 0
	for _, w := range workers {
		running, err := s.workerRepo.HasRunningJobs(w.ProjectID, string(w.ID))
		if err != nil {
			return removed, err
		}
		if running {
			continue
		}
		if err := s.workerRepo.Delete(w.ProjectID, w.ID); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func (s *Service) cascadeQueue(ctx context.Context, projectID id.Int, queueName string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	for {
		n, err := s.jobRepo.DeleteByQueue(projectID, queueName, s.batchSize())
		if err != nil {
			return err
		}
		if n == 0 {
			break
		}
	}

	if err := s.detachQueueFromWorkers(projectID, queueName); err != nil {
		return err
	}

	qn := domainQueue.NewName(queueName, projectID)
	if s.broker != nil {
		_ = s.broker.DeleteStream(qn.Subject())
		_ = s.broker.DeleteStream(qn.DLQSubject())
	}

	return s.queueRepo.Delete(projectID, queueName)
}

func (s *Service) cascadeTenant(ctx context.Context, tenantID id.Int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	queues, err := s.queueRepo.ListByProject(tenantID)
	if err != nil {
		return err
	}
	for _, q := range queues {
		if q.Status != domainQueue.StatusDeleting {
			q.Status = domainQueue.StatusDeleting
			_ = s.queueRepo.Update(&q)
		}
	}

	for {
		n, err := s.jobRepo.DeleteByProject(tenantID, s.batchSize())
		if err != nil {
			return err
		}
		if n == 0 {
			break
		}
	}

	workers, err := s.workerRepo.ListByProject(tenantID)
	if err != nil {
		return err
	}
	for _, w := range workers {
		if err := s.workerRepo.Delete(tenantID, w.ID); err != nil {
			return err
		}
	}

	for _, q := range queues {
		qn := domainQueue.NewName(q.Name, tenantID)
		if s.broker != nil {
			_ = s.broker.DeleteStream(qn.Subject())
			_ = s.broker.DeleteStream(qn.DLQSubject())
		}
		if err := s.queueRepo.Delete(tenantID, q.Name); err != nil {
			return err
		}
	}

	if s.apiKeyService != nil {
		keys, err := s.apiKeyService.ListByProjectID(tenantID)
		if err != nil {
			return err
		}
		for _, k := range keys {
			_ = s.apiKeyService.Delete(k.ID)
		}
	}

	_ = s.tenantRepo.DeleteLimits(tenantID)
	return s.tenantRepo.DeleteTenant(tenantID)
}

func (s *Service) detachQueueFromWorkers(projectID id.Int, queueName string) error {
	workers, err := s.workerRepo.ListByProject(projectID)
	if err != nil {
		return err
	}
	for i := range workers {
		w := &workers[i]
		updated := domainWorker.RemoveQueue(w.Queues, queueName)
		if updated == w.Queues {
			continue
		}
		w.Queues = updated
		w.UpdatedAt = clock.Now()
		if err := s.workerRepo.Update(w); err != nil {
			return err
		}
	}
	return nil
}
