package clock

import (
	"github.com/qpubio/qpub-server/internal/shared/timeparse"
	"time"
)

// UTCService implements the Service interface, ensuring all time operations are in UTC
type UTCService struct{}

// NewUTCService creates a new UTC service instance
func NewUTCService() *UTCService {
	return &UTCService{}
}

// Now returns the current time in UTC
func (s *UTCService) Now() time.Time {
	return time.Now().UTC()
}

// Parse parses a time string and returns it in UTC
// Leverages the existing timeparse package which already converts to UTC
func (s *UTCService) Parse(timeStr string) (time.Time, error) {
	return timeparse.Parse(timeStr)
}
