package realtime

// Repository defines the interface for project realtime stat persistence operations
type Repository interface {
	// Incr increments the value of a project realtime stat
	Incr(key Key) error

	// IncrBy increments the value of a project realtime stat by a given amount
	IncrBy(key Key, value int64) error

	// Decr decrements the value of a project realtime stat
	Decr(key Key) error

	// DecrBy decrements the value of a project realtime stat by a given amount
	DecrBy(key Key, value int64) error

	// Get retrieves a project realtime stat by its key
	Get(key Key) (int64, error)

	// Set sets the value of a project realtime stat (overwrites)
	Set(key Key, value int64) error

	// GetByPattern retrieves all project realtime stats that match the given pattern
	GetByPattern(pattern string) ([]string, error)

	// Reset resets the value of a project realtime stat
	Reset(key Key) error

	// ResetByPattern resets the value of all project realtime stats that match the given pattern
	ResetByPattern(pattern string) error

	// Batch operations for better performance
	// BatchIncr performs multiple increment operations
	BatchIncr(keys []Key) error

	// BatchIncrBy performs multiple increment by value operations
	BatchIncrBy(operations map[Key]int64) error

	// BatchDecr performs multiple safe decrement operations
	BatchDecr(keys []Key) error

	// BatchDecrBy performs multiple safe decrement by value operations
	BatchDecrBy(operations map[Key]int64) error

	// BatchReset performs multiple reset operations
	BatchReset(keys []Key) error
}
