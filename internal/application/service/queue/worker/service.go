package worker

import (
	projectLog "github.com/qpubio/qpub-server/internal/domain/project/log"
	logBroadcast "github.com/qpubio/qpub-server/internal/domain/project/log/broadcast"
	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	domainWorker "github.com/qpubio/qpub-server/internal/domain/queue/worker"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"gorm.io/gorm"
)

type Service struct {
	repository     domainWorker.Repository
	jobRepo        domainJob.Repository
	logger         logger.Service
	logBroadcaster logBroadcast.Service
}

func NewService(
	repository domainWorker.Repository,
	jobRepo domainJob.Repository,
	logger logger.Service,
	logBroadcaster logBroadcast.Service,
) domainWorker.Service {
	return &Service{
		repository:     repository,
		jobRepo:        jobRepo,
		logger:         logger,
		logBroadcaster: logBroadcaster,
	}
}

func (s *Service) Register(params domainWorker.CreateParams) (domainWorker.Worker, error) {
	w, err := domainWorker.Create(params)
	if err != nil {
		return domainWorker.Worker{}, err
	}
	if err := s.repository.Create(w); err != nil {
		return domainWorker.Worker{}, err
	}
	if s.logBroadcaster != nil {
		queueName := ""
		if len(params.Queues) > 0 {
			queueName = params.Queues[0]
		}
		event := projectLog.CreateQueueEvent(projectLog.CreateQueueEventParams{
			Message:    "Worker registered",
			QueueName:  queueName,
			WorkerID:   string(w.ID),
			WorkerName: w.Name,
		})
		if err := s.logBroadcaster.PublishLog(params.ProjectID, projectLog.EventQueueWorkerRegistered, *event); err != nil {
			s.logger.Warn(log.Queue, "Failed to publish worker registered log: %v", err)
		}
	}
	return *w, nil
}

func (s *Service) Heartbeat(projectID id.Int, workerID id.ULID) (domainWorker.Worker, error) {
	w, err := s.repository.FindByID(projectID, workerID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domainWorker.Worker{}, domainWorker.ErrNotFound
		}
		return domainWorker.Worker{}, err
	}
	w.Heartbeat()
	if err := s.repository.Update(w); err != nil {
		return domainWorker.Worker{}, err
	}

	// Sliding visibility lease: keep running jobs held by this worker from reclaiming.
	if s.jobRepo != nil {
		if _, err := s.jobRepo.ExtendLease(projectID, string(workerID), clock.Now()); err != nil {
			s.logger.Warn(log.Queue, "Failed to extend job leases worker=%s err=%v", workerID, err)
		}
	}

	return *w, nil
}

func (s *Service) Get(projectID id.Int, workerID id.ULID) (domainWorker.Worker, error) {
	w, err := s.repository.FindByID(projectID, workerID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domainWorker.Worker{}, domainWorker.ErrNotFound
		}
		return domainWorker.Worker{}, err
	}
	return *w, nil
}

func (s *Service) ListByProject(projectID id.Int) ([]domainWorker.Worker, error) {
	return s.repository.ListByProject(projectID)
}
