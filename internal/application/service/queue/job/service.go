package job

import (
	"encoding/json"

	projectLog "github.com/qpubio/qpub-server/internal/domain/project/log"
	logBroadcast "github.com/qpubio/qpub-server/internal/domain/project/log/broadcast"
	domainJob "github.com/qpubio/qpub-server/internal/domain/queue/job"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"gorm.io/gorm"
)

type Service struct {
	repository     domainJob.Repository
	logger         logger.Service
	logBroadcaster logBroadcast.Service
}

func NewService(
	repository domainJob.Repository,
	logger logger.Service,
	logBroadcaster logBroadcast.Service,
) domainJob.Service {
	return &Service{
		repository:     repository,
		logger:         logger,
		logBroadcaster: logBroadcaster,
	}
}

func (s *Service) Get(projectID id.Int, queueName string, jobID id.ULID) (domainJob.Job, error) {
	j, err := s.repository.FindByID(projectID, queueName, jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domainJob.Job{}, domainJob.ErrNotFound
		}
		return domainJob.Job{}, err
	}
	return *j, nil
}

func (s *Service) List(filter domainJob.ListFilter) ([]domainJob.Job, error) {
	return s.repository.List(filter)
}

func (s *Service) CountByStatus(projectID id.Int, queueName string) (map[domainJob.Status]int64, error) {
	statuses := []domainJob.Status{
		domainJob.StatusPending,
		domainJob.StatusScheduled,
		domainJob.StatusRunning,
		domainJob.StatusCompleted,
		domainJob.StatusFailed,
		domainJob.StatusCancelled,
		domainJob.StatusDLQ,
	}
	counts := make(map[domainJob.Status]int64, len(statuses))
	for _, status := range statuses {
		n, err := s.repository.CountByStatus(projectID, queueName, status)
		if err != nil {
			return nil, err
		}
		counts[status] = n
	}
	return counts, nil
}

func (s *Service) Cancel(projectID id.Int, queueName string, jobID id.ULID) (domainJob.Job, error) {
	j, err := s.repository.FindByID(projectID, queueName, jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domainJob.Job{}, domainJob.ErrNotFound
		}
		return domainJob.Job{}, err
	}

	switch j.Status {
	case domainJob.StatusCompleted, domainJob.StatusCancelled, domainJob.StatusDLQ:
		return domainJob.Job{}, domainJob.ErrInvalidTransition
	}

	j.MarkCancelled()
	if err := s.repository.Update(j); err != nil {
		return domainJob.Job{}, err
	}
	return *j, nil
}

func (s *Service) Retry(projectID id.Int, queueName string, jobID id.ULID) (domainJob.Job, error) {
	j, err := s.repository.FindByID(projectID, queueName, jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domainJob.Job{}, domainJob.ErrNotFound
		}
		return domainJob.Job{}, err
	}

	if j.Status != domainJob.StatusFailed && j.Status != domainJob.StatusDLQ {
		return domainJob.Job{}, domainJob.ErrInvalidTransition
	}

	j.Status = domainJob.StatusPending
	j.ScheduleAt = nil
	j.ErrorMessage = ""
	j.StartedAt = nil
	j.CompletedAt = nil
	j.WorkerID = ""

	if err := s.repository.Update(j); err != nil {
		return domainJob.Job{}, err
	}
	if s.logBroadcaster != nil {
		event := projectLog.CreateQueueEvent(projectLog.CreateQueueEventParams{
			Message:   "Job retried",
			QueueName: queueName,
			JobID:     &j.ID,
			Status:    string(j.Status),
			Attempt:   j.Attempt,
		})
		if err := s.logBroadcaster.PublishLog(projectID, projectLog.EventQueueJobRetried, *event); err != nil {
			s.logger.Warn(log.Queue, "Failed to publish job retried log: %v", err)
		}
	}
	return *j, nil
}

func (s *Service) UpdateProgress(projectID id.Int, queueName string, jobID id.ULID, metadata json.RawMessage) (domainJob.Job, error) {
	j, err := s.repository.FindByID(projectID, queueName, jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domainJob.Job{}, domainJob.ErrNotFound
		}
		return domainJob.Job{}, err
	}

	j.Metadata = metadata
	if err := s.repository.Update(j); err != nil {
		return domainJob.Job{}, err
	}
	return *j, nil
}
