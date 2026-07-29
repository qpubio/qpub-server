package infrastructure

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Time struct {
	// Timezone specifies the application's timezone
	Timezone string `env:"APP_TIMEZONE" envDefault:"UTC"`

	// ValidateTimezone ensures timezone is valid on startup
	ValidateTimezone bool `env:"APP_VALIDATE_TIMEZONE" envDefault:"true"`
}

func NewTime() (*Time, error) {
	cfg := &Time{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse time config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid time config: %w", err)
	}

	return cfg, nil
}

func (c *Time) Validate() error {
	if c.ValidateTimezone {
		if _, err := time.LoadLocation(c.Timezone); err != nil {
			return fmt.Errorf("invalid timezone %s: %w", c.Timezone, err)
		}

		if c.Timezone != "UTC" {
			return fmt.Errorf("only UTC timezone is supported, got: %s", c.Timezone)
		}
	}

	return nil
}
