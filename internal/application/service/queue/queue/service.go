package queue

import (
	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"gorm.io/gorm"
)

type Service struct {
	repository domainQueue.Repository
	logger     logger.Service
}

func NewService(repository domainQueue.Repository, logger logger.Service) domainQueue.Service {
	return &Service{
		repository: repository,
		logger:     logger,
	}
}

func (s *Service) Create(params domainQueue.CreateParams) (domainQueue.Queue, error) {
	q, err := domainQueue.Create(params)
	if err != nil {
		return domainQueue.Queue{}, err
	}

	_, err = s.repository.Create(q)
	if err != nil {
		return domainQueue.Queue{}, err
	}

	return s.Get(params.ProjectID, params.Name)
}

func (s *Service) Update(projectID id.Int, name string, params domainQueue.UpdateParams) (domainQueue.Queue, error) {
	current, err := s.repository.FindByProjectAndName(projectID, name)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domainQueue.Queue{}, domainQueue.ErrNotFound
		}
		return domainQueue.Queue{}, err
	}

	if err := current.Update(params); err != nil {
		return domainQueue.Queue{}, err
	}

	if err := s.repository.Update(current); err != nil {
		return domainQueue.Queue{}, err
	}

	return *current, nil
}

func (s *Service) Get(projectID id.Int, name string) (domainQueue.Queue, error) {
	q, err := s.repository.FindByProjectAndName(projectID, name)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domainQueue.Queue{}, domainQueue.ErrNotFound
		}
		return domainQueue.Queue{}, err
	}
	return *q, nil
}

func (s *Service) List(projectID id.Int) ([]domainQueue.Queue, error) {
	return s.repository.ListByProject(projectID)
}

func (s *Service) Ensure(params domainQueue.CreateParams) (domainQueue.Queue, error) {
	existing, err := s.repository.FindByProjectAndName(params.ProjectID, params.Name)
	if err == nil {
		return *existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return domainQueue.Queue{}, err
	}
	return s.Create(params)
}
