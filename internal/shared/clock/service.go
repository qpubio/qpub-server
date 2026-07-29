package clock

import (
	"sync"
	"time"
)

// Service provides centralized time operations, ensuring consistent UTC usage throughout the application
type Service interface {
	// Now returns the current time in UTC
	Now() time.Time

	// Parse parses a time string and returns it in UTC
	Parse(timeStr string) (time.Time, error)
}

var (
	service Service
	once    sync.Once
)

// Init initializes the clock service singleton with UTC service
func Init() {
	once.Do(func() {
		service = NewUTCService()
	})
}

// Get returns the clock service singleton instance
func Get() Service {
	if service == nil {
		panic("clock service not initialized - call clock.Init() first")
	}
	return service
}

// Convenience functions using the singleton service

// Now returns the current time in UTC
func Now() time.Time {
	return Get().Now()
}

// Parse parses a time string and returns it in UTC
func Parse(timeStr string) (time.Time, error) {
	return Get().Parse(timeStr)
}
