package duration

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Parse converts a string representation like "30d" or "5y" to time.Duration
func Parse(s string) (time.Duration, error) {
	// Handle standard Go duration format first
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Define regex pattern for custom duration format
	pattern := regexp.MustCompile(`^(\d+)([smhdwy])$`)
	matches := pattern.FindStringSubmatch(s)

	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	// Extract value and unit
	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid duration value: %s", matches[1])
	}

	// Convert to time.Duration based on unit
	unit := matches[2]
	switch unit {
	case "s":
		return time.Duration(value) * time.Second, nil
	case "m":
		return time.Duration(value) * time.Minute, nil
	case "h":
		return time.Duration(value) * time.Hour, nil
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "w":
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case "y":
		return time.Duration(value) * 365 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported duration unit: %s", unit)
	}
}

// Format converts a time.Duration to a human-readable string
func Format(d time.Duration) string {
	// Convert to seconds first
	seconds := int(d.Seconds())

	// Handle years
	if seconds >= 365*24*3600 {
		years := seconds / (365 * 24 * 3600)
		return fmt.Sprintf("%dy", years)
	}

	// Handle weeks
	if seconds >= 7*24*3600 {
		weeks := seconds / (7 * 24 * 3600)
		return fmt.Sprintf("%dw", weeks)
	}

	// Handle days
	if seconds >= 24*3600 {
		days := seconds / (24 * 3600)
		return fmt.Sprintf("%dd", days)
	}

	// Handle hours
	if seconds >= 3600 {
		hours := seconds / 3600
		return fmt.Sprintf("%dh", hours)
	}

	// Handle minutes
	if seconds >= 60 {
		minutes := seconds / 60
		return fmt.Sprintf("%dm", minutes)
	}

	// Handle seconds
	return fmt.Sprintf("%ds", seconds)
}

// MustParse is like Parse but panics on error
func MustParse(s string) time.Duration {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}
