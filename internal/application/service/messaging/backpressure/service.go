package backpressure

import (
	"github.com/qpubio/qpub-server/internal/domain/messaging/backpressure"
	"github.com/qpubio/qpub-server/internal/domain/tenant"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

// GatekeeperService enforces per-project inbound/outbound message rates.
type GatekeeperService struct {
	logger   logger.Service
	limits   backpressure.LimitsProvider
	inbound  *backpressure.ProjectLimiter
	outbound *backpressure.ProjectLimiter
}

// NewGatekeeperService creates a rate-limit gatekeeper.
func NewGatekeeperService(
	logger logger.Service,
	limits backpressure.LimitsProvider,
) backpressure.Gatekeeper {
	return &GatekeeperService{
		logger:   logger,
		limits:   limits,
		inbound:  backpressure.NewProjectLimiter(),
		outbound: backpressure.NewProjectLimiter(),
	}
}

func (g *GatekeeperService) AllowInbound(projectID id.Int) bool {
	return g.allow(projectID, true)
}

func (g *GatekeeperService) AllowOutbound(projectID id.Int) bool {
	return g.allow(projectID, false)
}

func (g *GatekeeperService) allow(projectID id.Int, inbound bool) bool {
	limits, err := g.limits.MessageRates(projectID)
	if err != nil {
		g.logger.Warn(log.MessagingPublication, "Failed to resolve message rate limits projectID=%d error=%v", projectID, err)
		return true
	}

	limit := limits.OutboundPerSecond
	limiter := g.outbound
	if inbound {
		limit = limits.InboundPerSecond
		limiter = g.inbound
	}

	if limit == tenant.Unlimited {
		return true
	}

	return limiter.Allow(projectID, limit)
}
