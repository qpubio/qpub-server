package period

import "time"

// Type represents a time period type
type Type string

// Available period types
const (
	Minutely Type = "minutely"
	Hourly   Type = "hourly"
	Daily    Type = "daily"
	Monthly  Type = "monthly"
	Yearly   Type = "yearly"
)

// TimeScale represents how much faster time moves in test mode
type TimeScale struct {
	Minutely time.Duration
	Hourly   time.Duration
	Daily    time.Duration
	Monthly  time.Duration
	Yearly   time.Duration
}

// Service handles period-based time operations
type Service struct {
	testMode  bool
	timeScale TimeScale
}

// DefaultTimeScale provides standard durations for production
var DefaultTimeScale = TimeScale{
	Minutely: 1 * time.Minute,      // 1 minute
	Hourly:   60 * time.Minute,     // 60 minutes
	Daily:    1440 * time.Minute,   // 24 hours = 1440 minutes
	Monthly:  43200 * time.Minute,  // 30 days = 43200 minutes
	Yearly:   525600 * time.Minute, // 365 days = 525600 minutes
}

// TestTimeScale provides shortened durations for testing while maintaining proper minute-based aggregation
// Base unit is 1 minute, scaled up by factors of 2 to maintain proper aggregation
var TestTimeScale = TimeScale{
	Minutely: 1 * time.Minute,  // 1 minute (base unit)
	Hourly:   2 * time.Minute,  // 2 minutes (aggregates 2 base minutes)
	Daily:    4 * time.Minute,  // 4 minutes (aggregates 2 hourly periods)
	Monthly:  8 * time.Minute,  // 8 minutes (aggregates 2 daily periods)
	Yearly:   16 * time.Minute, // 16 minutes (aggregates 2 monthly periods)
}

// QuickTestTimeScale provides very short durations while still maintaining proper aggregation
var QuickTestTimeScale = TimeScale{
	Minutely: 1 * time.Minute, // 1 minute (base unit)
	Hourly:   1 * time.Minute, // 1 minute (minimum for proper minute-based stats)
	Daily:    2 * time.Minute, // 2 minutes (aggregates 2 hourly periods)
	Monthly:  4 * time.Minute, // 4 minutes (aggregates 2 daily periods)
	Yearly:   8 * time.Minute, // 8 minutes (aggregates 2 monthly periods)
}

// NewService creates a new period service
func NewService(testMode bool) *Service {
	scale := DefaultTimeScale
	if testMode {
		scale = QuickTestTimeScale
	}
	return &Service{
		testMode:  testMode,
		timeScale: scale,
	}
}

// NewServiceWithScale creates a new period service with a custom time scale
func NewServiceWithScale(scale TimeScale) *Service {
	return &Service{
		testMode:  true,
		timeScale: scale,
	}
}

// NormalizeStartTime returns the normalized start time for a given period
func (s *Service) NormalizeStartTime(t time.Time, p Type) time.Time {
	if s.testMode {
		// In test mode, normalize to the period boundaries based on minutes
		totalMinutes := t.Hour()*60 + t.Minute()
		duration := s.GetPeriodDuration(p)
		minutePeriod := int(duration.Minutes())
		if minutePeriod <= 0 {
			minutePeriod = 1 // fallback to 1 minute if period is too small
		}

		// Round down to nearest period boundary
		normalizedMinutes := totalMinutes - (totalMinutes % minutePeriod)

		return time.Date(
			t.Year(), t.Month(), t.Day(),
			normalizedMinutes/60, // hours
			normalizedMinutes%60, // minutes
			0, 0, t.Location(),
		)
	}

	// Production mode uses calendar units
	switch p {
	case Minutely:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
	case Hourly:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	case Daily:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case Monthly:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case Yearly:
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	}
	return t
}

// CalculateEndTime returns the end time based on start time and period
func (s *Service) CalculateEndTime(startTime time.Time, p Type) time.Time {
	return startTime.Add(s.GetPeriodDuration(p))
}

// CalculateStartTimeOfPrevious returns the start time of the previous period
func (s *Service) CalculateStartTimeOfPrevious(startTime time.Time, p Type) time.Time {
	return startTime.Add(-s.GetPeriodDuration(p))
}

// GetNextSmallerPeriod returns the next smaller period type
func (s *Service) GetNextSmallerPeriod(p Type) Type {
	switch p {
	case Yearly:
		return Monthly
	case Monthly:
		return Daily
	case Daily:
		return Hourly
	case Hourly:
		return Minutely
	}
	return p // If already at smallest (Minutely), return same
}

// GetPeriodDuration returns the duration of the specified period type
func (s *Service) GetPeriodDuration(p Type) time.Duration {
	switch p {
	case Minutely:
		return s.timeScale.Minutely
	case Hourly:
		return s.timeScale.Hourly
	case Daily:
		return s.timeScale.Daily
	case Monthly:
		return s.timeScale.Monthly
	case Yearly:
		return s.timeScale.Yearly
	}
	return 0
}

// For testing purposes only
func (s *Service) GetTimeScale() TimeScale {
	return s.timeScale
}

// For testing purposes only
func (s *Service) SetTimeScale(scale TimeScale) {
	s.timeScale = scale
}
