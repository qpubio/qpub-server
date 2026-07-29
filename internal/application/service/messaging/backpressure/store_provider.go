package backpressure

import (
	"github.com/qpubio/qpub-server/internal/domain/messaging/backpressure"
	"github.com/qpubio/qpub-server/internal/domain/tenant"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// StoreLimitsProvider resolves message rates from the tenant limits store
// (populated by the control plane via PUT /control/v1/tenants/:id/limits).
type StoreLimitsProvider struct {
	tenantService tenant.Service
}

func NewStoreLimitsProvider(tenantService tenant.Service) backpressure.LimitsProvider {
	return &StoreLimitsProvider{tenantService: tenantService}
}

func (p *StoreLimitsProvider) MessageRates(projectID id.Int) (backpressure.MessageRateLimits, error) {
	l, err := p.tenantService.GetLimits(projectID)
	if err != nil {
		return backpressure.MessageRateLimits{}, err
	}
	return backpressure.MessageRateLimits{
		InboundPerSecond:  l.InboundPerSecond,
		OutboundPerSecond: l.OutboundPerSecond,
	}, nil
}
