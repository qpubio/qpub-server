package realtime

import "github.com/qpubio/qpub-server/internal/shared/id"

// Service defines the interface for project realtime stat operations
type Service interface {
	// Increment operations
	// Incr increments the value of a project realtime stat
	Incr(key Key) error

	// IncrBy increments the value of a project realtime stat by a given amount
	IncrBy(key Key, value int64) error

	// Decrement operations
	// Decr decrements the value of a project realtime stat
	Decr(key Key) error

	// DecrBy decrements the value of a project realtime stat by a given amount
	DecrBy(key Key, value int64) error

	// Retrieval operations
	// Get retrieves a project realtime stat by its key
	Get(key Key) (int64, error)

	// GetByPattern retrieves all project realtime stats that match the given pattern
	GetByPattern(pattern string) (map[string]int64, error)

	// Set operations
	// Set sets the value of a stat (overwrites)
	Set(key Key, value int64) error

	// Reset operations
	// Reset resets the value of a project realtime stat
	Reset(key Key) error

	// ResetByPattern resets the value of all project realtime stats that match the given pattern
	ResetByPattern(pattern string) error

	// Additional aggregation operations
	// GetSummary retrieves a summary of all stats for a project
	GetSummary(projID id.Int) (map[string]int64, error)
}
