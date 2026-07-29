package tenant

import (
	"github.com/qpubio/qpub-server/internal/shared/id"
	"time"
)

// Tenant is the messaging/queue isolation unit.
type Tenant struct {
	ID        id.Int `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Tenant) TableName() string {
	return "tenants"
}

// Limits is the enforced rate-limit snapshot for a tenant (set via the control API).
type Limits struct {
	TenantID          id.Int `gorm:"primarykey"`
	InboundPerSecond  int64  `gorm:"not null;default:-1"`
	OutboundPerSecond int64  `gorm:"not null;default:-1"`
	UpdatedAt         time.Time
}

func (Limits) TableName() string {
	return "tenant_limits"
}

const Unlimited int64 = -1
