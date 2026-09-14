package worker

import (
	projectLog "github.com/qpubio/qpub-server/internal/domain/project/log"
	logBroadcast "github.com/qpubio/qpub-server/internal/domain/project/log/broadcast"
	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	"github.com/qpubio/qpub-server/internal/domain/queue/lifecycle"
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
	guard          *lifecycle.Guard
}

func NewService(
	repository domainWorker.Repository,
	jobRepo domainJob.Repository,
	logger logger.Service,
	logBroadcaster logBroadcast.Service,
	guard *lifecycle.Guard,
) domainWorker.Service {
	return &Service{
		repository:     repository,
		jobRepo:        jobRepo,
		logger:         logger,
		logBroadcaster: logBroadcaster,
		guard:          guard,
	}
}

func (s *Service) Register(params domainWorker.CreateParams) (domainWorker.Worker, error) {
	if s.guard != nil {
		if err := s.guard.AssertTenantWritable(params.ProjectID); err != nil {
			return domainWorker.Worker{}, err
		}
		for _, q := range params.Queues {
			if err := s.guard.AssertQueueWritable(params.ProjectID, q); err != nil {
				return domainWorker.Worker{}, err
			}
		}
	}
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

func (s *Service) ListByProjectPaginated(projectID id.Int, page, perPage int) ([]domainWorker.Worker, int64, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 10
	}
	return s.repository.ListByProjectPaginated(projectID, perPage, (page-1)*perPage)
}
