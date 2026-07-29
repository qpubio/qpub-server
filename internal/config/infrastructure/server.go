package infrastructure

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Server struct {
	AdminPort     string `env:"PORT_ADMIN_API" envDefault:"8081"`
	ControlPort   string `env:"PORT_CONTROL_API" envDefault:"8091"`
	RestPort      string `env:"PORT_REST_API" envDefault:"8111"`
	WebSocketPort string `env:"PORT_WEBSOCKET" envDefault:"8131"`
}

func NewServer() (*Server, error) {
	cfg := &Server{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse server config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid server config: %w", err)
	}
	return cfg, nil
}

func (s *Server) Validate() error {
	if s.AdminPort == "" {
		return fmt.Errorf("admin port is required")
	}
	if s.ControlPort == "" {
		return fmt.Errorf("control port is required")
	}
	if s.RestPort == "" {
		return fmt.Errorf("rest port is required")
	}
	if s.WebSocketPort == "" {
		return fmt.Errorf("websocket port is required")
	}
	return nil
}
