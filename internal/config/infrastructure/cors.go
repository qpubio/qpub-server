package infrastructure

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type CORS struct {
	Admin     string `env:"CORS_ADMIN_ORIGINS" envDefault:"*"`
	Control   string `env:"CORS_CONTROL_ORIGINS" envDefault:"*"`
	Dashboard string `env:"CORS_DASHBOARD_ORIGINS" envDefault:"*"`
	Rest      string `env:"CORS_REST_ORIGINS" envDefault:"*"`
	Webhook   string `env:"CORS_WEBHOOK_ORIGINS" envDefault:"*"`
	WebSocket string `env:"CORS_WEBSOCKET_ORIGINS" envDefault:"*"`
}

func NewCORS() (*CORS, error) {
	cors := &CORS{}
	if err := env.Parse(cors); err != nil {
		return nil, fmt.Errorf("failed to parse cors config: %w", err)
	}

	if err := cors.Validate(); err != nil {
		return nil, fmt.Errorf("invalid cors config: %w", err)
	}

	return cors, nil
}

func (c *CORS) Validate() error {
	if c.Admin == "" {
		return fmt.Errorf("admin cors origins is required")
	}

	if c.Control == "" {
		return fmt.Errorf("control cors origins is required")
	}

	if c.Dashboard == "" {
		return fmt.Errorf("dashboard cors origins is required")
	}

	if c.Rest == "" {
		return fmt.Errorf("rest cors origins is required")
	}

	if c.Webhook == "" {
		return fmt.Errorf("webhook cors origins is required")
	}

	if c.WebSocket == "" {
		return fmt.Errorf("websocket cors origins is required")
	}

	return nil
}
