package maintenance

import (
	"context"

	domainCleanup "github.com/qpubio/qpub-server/internal/domain/queue/cleanup"
)

// Service runs operational queue maintenance tasks.
type Service struct {
	cleanup domainCleanup.Service
}

func NewService(cleanup domainCleanup.Service) *Service {
	return &Service{cleanup: cleanup}
}

func (s *Service) PurgeStaleWorkers(ctx context.Context) (int, error) {
	return s.cleanup.PurgeStaleWorkers(ctx)
}

func (s *Service) ResumePendingCascades(ctx context.Context) error {
	return s.cleanup.ResumePendingCascades(ctx)
}
