package backpressure

import "github.com/qpubio/qpub-server/internal/shared/id"

// LimitsProvider resolves message rate limits for a project.
type LimitsProvider interface {
	MessageRates(projectID id.Int) (MessageRateLimits, error)
}

// Gatekeeper enforces project message rate limits.
type Gatekeeper interface {
	AllowInbound(projectID id.Int) bool
	AllowOutbound(projectID id.Int) bool
}
