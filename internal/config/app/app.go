package app

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// App holds process-level configuration for the data-plane server.
type App struct {
	Env   string `env:"APP_ENV" envDefault:"production"`
	Debug bool   `env:"APP_DEBUG" envDefault:"false"`

	// ControlAPIToken authenticates the control API used to provision tenants/keys/limits.
	ControlAPIToken string `env:"CONTROL_API_TOKEN" envDefault:""`
}

func NewApp() (*App, error) {
	cfg := &App{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse app config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid app config: %w", err)
	}

	return cfg, nil
}

func (c *App) Validate() error {
	if !isValidEnvironment(c.Env) {
		return fmt.Errorf("invalid environment: %s", c.Env)
	}
	return nil
}

func isValidEnvironment(env string) bool {
	switch env {
	case "production", "development", "testing":
		return true
	default:
		return false
	}
}
