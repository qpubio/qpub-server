package tenant

import (
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Service manages messaging tenants and their pushed limit snapshots.
type Service interface {
	Ensure(tenantID id.Int) (Tenant, error)
	Delete(tenantID id.Int) error
	Get(tenantID id.Int) (Tenant, error)

	SetLimits(tenantID id.Int, inboundPerSecond, outboundPerSecond int64) (Limits, error)
	GetLimits(tenantID id.Int) (Limits, error)
}

// Repository persists tenants and limits.
type Repository interface {
	UpsertTenant(t Tenant) error
	DeleteTenant(tenantID id.Int) error
	FindTenant(tenantID id.Int) (*Tenant, error)

	UpsertLimits(l Limits) error
	FindLimits(tenantID id.Int) (*Limits, error)
	DeleteLimits(tenantID id.Int) error
}
