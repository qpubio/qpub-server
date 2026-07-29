package tenant

import (
	"fmt"
	"sync"
	"time"

	"github.com/qpubio/qpub-server/internal/domain/tenant"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

// Service implements tenant.Service with an in-memory store backed by optional repository.
type Service struct {
	logger logger.Service
	repo   tenant.Repository

	mu      sync.RWMutex
	tenants map[id.Int]tenant.Tenant
	limits  map[id.Int]tenant.Limits
}

func NewService(logger logger.Service, repo tenant.Repository) tenant.Service {
	return &Service{
		logger:  logger,
		repo:    repo,
		tenants: make(map[id.Int]tenant.Tenant),
		limits:  make(map[id.Int]tenant.Limits),
	}
}

func (s *Service) Ensure(tenantID id.Int) (tenant.Tenant, error) {
	if tenantID <= 0 {
		return tenant.Tenant{}, fmt.Errorf("invalid tenant id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.tenants[tenantID]; ok {
		return t, nil
	}

	now := time.Now().UTC()
	t := tenant.Tenant{ID: tenantID, CreatedAt: now, UpdatedAt: now}
	if s.repo != nil {
		if err := s.repo.UpsertTenant(t); err != nil {
			return tenant.Tenant{}, err
		}
	}
	s.tenants[tenantID] = t
	s.logger.Info(log.App, "Ensured messaging tenant id=%d", tenantID)
	return t, nil
}

func (s *Service) Delete(tenantID id.Int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.repo != nil {
		_ = s.repo.DeleteLimits(tenantID)
		if err := s.repo.DeleteTenant(tenantID); err != nil {
			return err
		}
	}
	delete(s.tenants, tenantID)
	delete(s.limits, tenantID)
	return nil
}

func (s *Service) Get(tenantID id.Int) (tenant.Tenant, error) {
	s.mu.RLock()
	if t, ok := s.tenants[tenantID]; ok {
		s.mu.RUnlock()
		return t, nil
	}
	s.mu.RUnlock()

	if s.repo != nil {
		t, err := s.repo.FindTenant(tenantID)
		if err != nil {
			return tenant.Tenant{}, err
		}
		if t != nil {
			s.mu.Lock()
			s.tenants[tenantID] = *t
			s.mu.Unlock()
			return *t, nil
		}
	}
	return tenant.Tenant{}, fmt.Errorf("tenant not found")
}

func (s *Service) SetLimits(tenantID id.Int, inboundPerSecond, outboundPerSecond int64) (tenant.Limits, error) {
	if _, err := s.Ensure(tenantID); err != nil {
		return tenant.Limits{}, err
	}

	l := tenant.Limits{
		TenantID:          tenantID,
		InboundPerSecond:  inboundPerSecond,
		OutboundPerSecond: outboundPerSecond,
		UpdatedAt:         time.Now().UTC(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.repo != nil {
		if err := s.repo.UpsertLimits(l); err != nil {
			return tenant.Limits{}, err
		}
	}
	s.limits[tenantID] = l
	return l, nil
}

func (s *Service) GetLimits(tenantID id.Int) (tenant.Limits, error) {
	s.mu.RLock()
	if l, ok := s.limits[tenantID]; ok {
		s.mu.RUnlock()
		return l, nil
	}
	s.mu.RUnlock()

	if s.repo != nil {
		l, err := s.repo.FindLimits(tenantID)
		if err != nil {
			return tenant.Limits{}, err
		}
		if l != nil {
			s.mu.Lock()
			s.limits[tenantID] = *l
			s.mu.Unlock()
			return *l, nil
		}
	}

	// Default: unlimited until control plane pushes a snapshot.
	return tenant.Limits{
		TenantID:          tenantID,
		InboundPerSecond:  tenant.Unlimited,
		OutboundPerSecond: tenant.Unlimited,
	}, nil
}
