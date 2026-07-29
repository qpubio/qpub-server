package backpressure

// MessageRateLimits holds plan-derived message rate limits for a project.
type MessageRateLimits struct {
	InboundPerSecond  int64
	OutboundPerSecond int64
}
