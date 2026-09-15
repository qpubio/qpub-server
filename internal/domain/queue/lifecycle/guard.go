package lifecycle

import (
	"errors"

	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
	"github.com/qpubio/qpub-server/internal/domain/tenant"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"gorm.io/gorm"
)

var (
	ErrTenantDeleting = errors.New("tenant is deleting")
	ErrQueueDeleting  = errors.New("queue is deleting")
)

// Guard enforces DELETING write barriers for tenants and queues.
type Guard struct {
	tenantRepo tenant.Repository
	queueRepo  domainQueue.Repository
}

func NewGuard(tenantRepo tenant.Repository, queueRepo domainQueue.Repository) *Guard {
	return &Guard{
		tenantRepo: tenantRepo,
		queueRepo:  queueRepo,
	}
}

func (g *Guard) AssertTenantWritable(projectID id.Int) error {
	if projectID <= 0 || g.tenantRepo == nil {
		return nil
	}
	t, err := g.tenantRepo.FindTenant(projectID)
	if err != nil {
		return err
	}
	if t == nil {
		return nil
	}
	if !t.Status.IsActive() {
		return ErrTenantDeleting
	}
	return nil
}

func (g *Guard) AssertQueueWritable(projectID id.Int, queueName string) error {
	if err := g.AssertTenantWritable(projectID); err != nil {
		return err
	}
	if g.queueRepo == nil || queueName == "" {
		return nil
	}
	q, err := g.queueRepo.FindByProjectAndName(projectID, queueName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if q != nil && !q.Status.IsActive() {
		return ErrQueueDeleting
	}
	return nil
}
