package duration

import (
	"time"
)

// DurationVar is a helper type for custom environment variables
type DurationVar struct {
	time.Duration
}

// UnmarshalText implements encoding.TextUnmarshaler
func (d *DurationVar) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = Parse(string(text))
	return err
}
