package lifecycle

import (
	"errors"
	"testing"
	"time"

	domainQueue "github.com/qpubio/qpub-server/internal/domain/queue/queue"
	"github.com/qpubio/qpub-server/internal/domain/tenant"
	"github.com/qpubio/qpub-server/internal/shared/id"

	"gorm.io/gorm"
)

type mockTenantRepo struct {
	tenant *tenant.Tenant
	err    error
}

func (m *mockTenantRepo) UpsertTenant(t tenant.Tenant) error                     { return nil }
func (m *mockTenantRepo) UpdateTenant(t tenant.Tenant) error                     { return nil }
func (m *mockTenantRepo) DeleteTenant(tenantID id.Int) error                     { return nil }
func (m *mockTenantRepo) ListDeleting(limit int) ([]tenant.Tenant, error)        { return nil, nil }
func (m *mockTenantRepo) UpsertLimits(l tenant.Limits) error                     { return nil }
func (m *mockTenantRepo) FindLimits(tenantID id.Int) (*tenant.Limits, error)     { return nil, nil }
func (m *mockTenantRepo) DeleteLimits(tenantID id.Int) error                     { return nil }

func (m *mockTenantRepo) FindTenant(tenantID id.Int) (*tenant.Tenant, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tenant, nil
}

type mockQueueRepo struct {
	queue *domainQueue.Queue
	err   error
}

func (m *mockQueueRepo) Create(queue *domainQueue.Queue) (id.Int, error) { return 0, nil }
func (m *mockQueueRepo) Update(queue *domainQueue.Queue) error         { return nil }
func (m *mockQueueRepo) Delete(projectID id.Int, name string) error    { return nil }
func (m *mockQueueRepo) FindByID(queueID id.Int) (*domainQueue.Queue, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockQueueRepo) ListByProjectPaginated(projectID id.Int, limit, offset int) ([]domainQueue.Queue, int64, error) {
	return nil, 0, nil
}
func (m *mockQueueRepo) ListByProject(projectID id.Int) ([]domainQueue.Queue, error) {
	return nil, nil
}
func (m *mockQueueRepo) ListDeleting(limit int) ([]domainQueue.Queue, error) { return nil, nil }

func (m *mockQueueRepo) FindByProjectAndName(projectID id.Int, name string) (*domainQueue.Queue, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.queue, nil
}

func TestAssertQueueWritable_NotFound(t *testing.T) {
	g := NewGuard(&mockTenantRepo{}, &mockQueueRepo{err: gorm.ErrRecordNotFound})
	if err := g.AssertQueueWritable(1, "my-queue"); err != nil {
		t.Fatalf("expected nil for missing queue, got %v", err)
	}
}

func TestAssertQueueWritable_Active(t *testing.T) {
	g := NewGuard(&mockTenantRepo{}, &mockQueueRepo{
		queue: &domainQueue.Queue{
			ProjectID: 1,
			Name:      "my-queue",
			Status:    domainQueue.StatusActive,
		},
	})
	if err := g.AssertQueueWritable(1, "my-queue"); err != nil {
		t.Fatalf("expected nil for active queue, got %v", err)
	}
}

func TestAssertQueueWritable_Deleting(t *testing.T) {
	g := NewGuard(&mockTenantRepo{}, &mockQueueRepo{
		queue: &domainQueue.Queue{
			ProjectID: 1,
			Name:      "my-queue",
			Status:    domainQueue.StatusDeleting,
		},
	})
	if err := g.AssertQueueWritable(1, "my-queue"); !errors.Is(err, ErrQueueDeleting) {
		t.Fatalf("expected ErrQueueDeleting, got %v", err)
	}
}

func TestAssertQueueWritable_TenantDeleting(t *testing.T) {
	g := NewGuard(&mockTenantRepo{
		tenant: &tenant.Tenant{
			ID:     1,
			Status: tenant.StatusDeleting,
		},
	}, &mockQueueRepo{})
	if err := g.AssertQueueWritable(1, "my-queue"); !errors.Is(err, ErrTenantDeleting) {
		t.Fatalf("expected ErrTenantDeleting, got %v", err)
	}
}

func TestAssertTenantWritable_MissingTenant(t *testing.T) {
	g := NewGuard(&mockTenantRepo{tenant: nil}, &mockQueueRepo{})
	if err := g.AssertTenantWritable(1); err != nil {
		t.Fatalf("expected nil for missing tenant row, got %v", err)
	}
}

func TestAssertTenantWritable_Active(t *testing.T) {
	g := NewGuard(&mockTenantRepo{
		tenant: &tenant.Tenant{
			ID:        1,
			Status:    tenant.StatusActive,
			CreatedAt: time.Now(),
		},
	}, &mockQueueRepo{})
	if err := g.AssertTenantWritable(1); err != nil {
		t.Fatalf("expected nil for active tenant, got %v", err)
	}
}
