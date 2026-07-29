package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainRouter "github.com/qpubio/qpub-server/internal/domain/queue/router"
	domainRuntime "github.com/qpubio/qpub-server/internal/domain/queue/runtime"
	"github.com/qpubio/qpub-server/internal/application/service/queue/dispatch/webhook"
	"github.com/qpubio/qpub-server/internal/application/service/queue/platform"
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

type Service struct {
	router      domainRouter.Service
	jobRepo     domainJob.Repository
	queueRepo   domainQueue.Repository
	webhook     *webhook.Service
	platformReg *platform.Registry
	logger      logger.Service
	instanceID  string

	handlers map[string]domainRuntime.JobHandler
	mu       sync.RWMutex

	runCtx   context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}
	wg       sync.WaitGroup
	started  bool
}

func NewService(
	router domainRouter.Service,
	jobRepo domainJob.Repository,
	queueRepo domainQueue.Repository,
	webhookSvc *webhook.Service,
	platformReg *platform.Registry,
	logger logger.Service,
	instanceID id.ULID,
) domainRuntime.Service {
	return &Service{
		router:      router,
		jobRepo:     jobRepo,
		queueRepo:   queueRepo,
		webhook:     webhookSvc,
		platformReg: platformReg,
		logger:      logger,
		instanceID:  string(instanceID),
		handlers:    make(map[string]domainRuntime.JobHandler),
	}
}

func (s *Service) RegisterHandler(queueName string, handler domainRuntime.JobHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[queueName] = handler
}

func (s *Service) Start(ctx context.Context) error {
	if s.started {
		return nil
	}
	s.runCtx, s.cancel = context.WithCancel(ctx)
	s.stopChan = make(chan struct{})
	s.wg.Add(1)
	go s.workerLoop(s.runCtx)
	s.started = true
	s.logger.Info(log.Queue, "Queue runtime started")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if !s.started {
		return nil
	}
	if s.cancel != nil {
		s.cancel()
	}
	close(s.stopChan)
	s.wg.Wait()
	s.started = false
	s.logger.Info(log.Queue, "Queue runtime stopped")
	return nil
}

func (s *Service) workerLoop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.processPlatformQueues(ctx)
			s.processCustomQueues(ctx)
		}
	}
}

func (s *Service) processPlatformQueues(ctx context.Context) {
	for _, def := range s.platformReg.All() {
		queueName := platform.QueueName(def.Name)
		s.processQueue(ctx, platform.PlatformProjectID, string(queueName))
	}
}

func (s *Service) processCustomQueues(ctx context.Context) {
	// Custom queue workers pull via REST API; runtime handles webhook dispatch only.
}

func (s *Service) processQueue(ctx context.Context, projectID id.Int, queueName string) {
	s.mu.RLock()
	handler, ok := s.handlers[queueName]
	s.mu.RUnlock()
	if !ok {
		return
	}

	jobs, err := s.router.Dequeue(ctx, domainJob.DequeueRequest{
		ProjectID: projectID,
		QueueName: queueName,
		WorkerID:  s.instanceID,
		BatchSize: 1,
	})
	if err != nil || len(jobs) == 0 {
		return
	}

	for _, job := range jobs {
		s.executeJob(ctx, projectID, queueName, handler, job)
	}
}

func (s *Service) executeJob(ctx context.Context, projectID id.Int, queueName string, handler domainRuntime.JobHandler, job domainJob.Job) {
	defer func() {
		if recovered := recover(); recovered != nil {
			reason := fmt.Sprintf("panic: %v", recovered)
			s.logger.Error(log.Queue, "Job handler panicked job=%s err=%v", job.ID, recovered)
			_, nackErr := s.router.Nack(ctx, domainJob.NackRequest{
				ProjectID: projectID,
				QueueName: queueName,
				JobID:     job.ID,
				WorkerID:  s.instanceID,
				Reason:    reason,
			})
			if nackErr != nil {
				s.logger.Error(log.Queue, "Failed to nack panicked job job=%s err=%v", job.ID, nackErr)
			}
		}
	}()

	if q, err := s.queueRepo.FindByProjectAndName(projectID, queueName); err == nil && q.WebhookURL != "" {
		if err := s.webhook.Dispatch(ctx, *q, job); err != nil {
			s.logger.Warn(log.Queue, "Webhook dispatch failed job=%s err=%v", job.ID, err)
		}
	}

	if err := handler(ctx, job.Payload); err != nil {
		reason := err.Error()
		_, nackErr := s.router.Nack(ctx, domainJob.NackRequest{
			ProjectID: projectID,
			QueueName: queueName,
			JobID:     job.ID,
			WorkerID:  s.instanceID,
			Reason:    reason,
		})
		if nackErr != nil {
			s.logger.Error(log.Queue, "Failed to nack job job=%s err=%v", job.ID, nackErr)
		}
		return
	}

	result, _ := json.Marshal(map[string]string{"status": "ok"})
	_, err := s.router.Ack(ctx, domainJob.AckRequest{
		ProjectID: projectID,
		QueueName: queueName,
		JobID:     job.ID,
		WorkerID:  s.instanceID,
		Result:    result,
	})
	if err != nil {
		s.logger.Error(log.Queue, "Failed to ack job job=%s err=%v", job.ID, err)
	}
}
