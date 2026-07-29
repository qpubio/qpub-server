package backpressure

import (
	domainBackpressure "github.com/qpubio/qpub-server/internal/domain/queue/backpressure"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

type Service struct{}

func NewService() domainBackpressure.Gatekeeper {
	return &Service{}
}

func (s *Service) AllowEnqueue(projectID id.Int) (bool, error) {
	_ = projectID
	return true, nil
}

func (s *Service) AllowDequeue(projectID id.Int) (bool, error) {
	_ = projectID
	return true, nil
}
