package apikey

import (
	"github.com/qpubio/qpub-server/internal/domain/apikey"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

type Service struct {
	logger     logger.Service
	repository apikey.Repository
}

func NewService(
	logger logger.Service,
	repository apikey.Repository,
) apikey.Service {
	return &Service{
		logger:     logger,
		repository: repository,
	}
}

func (s *Service) Create(params apikey.CreateParams) (apikey.APIKey, error) {
	apik, err := apikey.Create(apikey.CreateParams{
		ProjectID:  params.ProjectID,
		Name:       params.Name,
		Permission: params.Permission,
		Metadata:   params.Metadata,
		Status:     params.Status,
		ExpiresAt:  params.ExpiresAt,
	})
	if err != nil {
		return apikey.APIKey{}, err
	}

	akID, err := s.repository.Create(apik)
	if err != nil {
		return apikey.APIKey{}, err
	}

	return s.Get(akID)
}

func (s *Service) Update(akID id.Int, params apikey.UpdateParams) (apikey.APIKey, error) {
	apik, err := s.Get(akID)
	if err != nil {
		return apikey.APIKey{}, err
	}

	err = apik.Update(apikey.UpdateParams{
		Name:       params.Name,
		Permission: params.Permission,
		Metadata:   params.Metadata,
		Status:     params.Status,
		LastUsedAt: params.LastUsedAt,
		ExpiresAt:  params.ExpiresAt,
	})
	if err != nil {
		return apikey.APIKey{}, err
	}

	err = s.repository.Update(&apik)
	if err != nil {
		return apikey.APIKey{}, err
	}

	return s.Get(akID)
}

func (s *Service) Delete(akID id.Int) error {
	ak, err := s.Get(akID)
	if err != nil {
		return err
	}

	return s.repository.Delete(&ak)
}

func (s *Service) Get(akID id.Int) (apikey.APIKey, error) {
	apik, err := s.repository.FindByID(akID)
	if err != nil {
		return apikey.APIKey{}, err
	}

	return *apik, nil
}

func (s *Service) GetByIDs(akIDs []id.Int) ([]apikey.APIKey, error) {
	apik, err := s.repository.FindByIDs(akIDs)
	if err != nil {
		return []apikey.APIKey{}, err
	}

	return apik, nil
}

func (s *Service) ListByProjectID(projID id.Int) ([]apikey.APIKey, error) {
	return s.repository.ListByProjectID(projID)
}
