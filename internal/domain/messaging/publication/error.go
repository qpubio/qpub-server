package publication

import "errors"

// ErrRateLimited is returned when a publish exceeds plan inbound rate limits.
var ErrRateLimited = errors.New("rate limit exceeded")
