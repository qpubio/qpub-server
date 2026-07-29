package backpressure

import (
	"sync"
	"time"

	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

const defaultWindow = time.Second

// FixedWindowLimiter enforces a per-second count limit.
type FixedWindowLimiter struct {
	mu     sync.Mutex
	limit  int64
	window time.Duration
	count  int64
	start  time.Time
}

// NewFixedWindowLimiter creates a limiter with the given per-window limit.
func NewFixedWindowLimiter(limit int64) *FixedWindowLimiter {
	if limit < 0 {
		limit = 0
	}
	return &FixedWindowLimiter{
		limit:  limit,
		window: defaultWindow,
		start:  clock.Now(),
	}
}

// Limit returns the configured limit.
func (l *FixedWindowLimiter) Limit() int64 {
	if l == nil {
		return 0
	}
	return l.limit
}

// Allow reports whether another event is permitted in the current window.
func (l *FixedWindowLimiter) Allow() bool {
	if l == nil || l.limit <= 0 {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := clock.Now()
	if now.Sub(l.start) >= l.window {
		l.start = now
		l.count = 1
		return true
	}

	if l.count >= l.limit {
		return false
	}

	l.count++
	return true
}

// ProjectLimiter tracks independent fixed-window limiters per project.
type ProjectLimiter struct {
	mu       sync.Mutex
	limiters map[id.Int]*FixedWindowLimiter
}

// NewProjectLimiter creates an empty per-project limiter registry.
func NewProjectLimiter() *ProjectLimiter {
	return &ProjectLimiter{
		limiters: make(map[id.Int]*FixedWindowLimiter),
	}
}

// Allow checks the limit for a project, creating or refreshing the limiter when needed.
func (p *ProjectLimiter) Allow(projectID id.Int, limit int64) bool {
	if limit <= 0 {
		return true
	}

	p.mu.Lock()
	lim := p.limiters[projectID]
	if lim == nil || lim.Limit() != limit {
		lim = NewFixedWindowLimiter(limit)
		p.limiters[projectID] = lim
	}
	p.mu.Unlock()

	return lim.Allow()
}
