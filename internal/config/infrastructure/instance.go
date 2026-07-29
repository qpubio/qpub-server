package infrastructure

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Instance struct {
	// Heartbeat is used to check if the instance is still running
	Heartbeat struct {
		// Interval is the interval in seconds to check if the instance is still running
		Interval int `env:"INSTANCE_HEARTBEAT_INTERVAL" envDefault:"30"` // seconds
	}

	// InactivityTimeout is the timeout in seconds to consider the instance as inactive
	InactivityTimeout int `env:"INSTANCE_INACTIVITY_TIMEOUT" envDefault:"90"` // seconds
}

func NewInstance() (*Instance, error) {
	cfg := &Instance{}

	// Parse environment variables
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse instance config: %w", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate instance config: %w", err)
	}

	return cfg, nil
}

func (i *Instance) Validate() error {
	if i.Heartbeat.Interval <= 0 {
		return fmt.Errorf("heartbeat interval must be greater than 0")
	}

	if i.InactivityTimeout <= 0 {
		return fmt.Errorf("inactivity timeout must be greater than 0")
	}

	return nil
}
