package timeparse

import (
	"strconv"
	"time"
)

// Parse parses a time parameter from various common formats
// Supports Unix timestamps (seconds/milliseconds), RFC3339, date-time, and date formats
func Parse(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, nil
	}

	// List of time formats to try, in order of preference
	formats := []string{
		// Unix timestamps (try these first as they're unambiguous)
		"unix_seconds",
		"unix_milliseconds",
		// RFC3339 formats
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z0700",
		"2006-01-02T15:04:05",
		// Common date-time formats
		"2006-01-02 15:04:05",
		"2006-01-02-15:04:05",
		"2006-01-02 15:04:05+00",
		"2006-01-02 15:04:05 MST",
		// Date only formats (will use start of day)
		"2006-01-02",
		"2006/01/02",
		"01/02/2006",
	}

	for _, format := range formats {
		var t time.Time
		var err error

		switch format {
		case "unix_seconds":
			// Try parsing as Unix timestamp (seconds)
			if timestamp, parseErr := strconv.ParseInt(timeStr, 10, 64); parseErr == nil {
				// Validate reasonable timestamp range (between 2000-01-01 and 2100-01-01)
				if timestamp >= 946684800 && timestamp <= 4102444800 {
					t = time.Unix(timestamp, 0).UTC()
					return t, nil
				}
			}
		case "unix_milliseconds":
			// Try parsing as Unix timestamp (milliseconds)
			if timestamp, parseErr := strconv.ParseInt(timeStr, 10, 64); parseErr == nil {
				// Validate reasonable timestamp range
				if timestamp >= 946684800000 && timestamp <= 4102444800000 {
					t = time.Unix(timestamp/1000, (timestamp%1000)*1000000).UTC()
					return t, nil
				}
			}
		default:
			t, err = time.Parse(format, timeStr)
			if err == nil {
				// Convert to UTC for consistency
				return t.UTC(), nil
			}
		}
	}

	// If none of the formats worked, return a descriptive error
	return time.Time{}, &time.ParseError{
		Layout:  "multiple formats tried",
		Value:   timeStr,
		Message: "unable to parse time. Supported formats: Unix timestamp (seconds/milliseconds), RFC3339 (2006-01-02T15:04:05Z), date-time (2006-01-02 15:04:05), date (2006-01-02)",
	}
}

// MustParse is like Parse but panics on error
func MustParse(timeStr string) time.Time {
	t, err := Parse(timeStr)
	if err != nil {
		panic(err)
	}
	return t
}
