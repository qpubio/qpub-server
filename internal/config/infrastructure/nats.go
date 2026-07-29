package infrastructure

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type NATS struct {
	URL      string `env:"NATS_URL" envDefault:"nats://localhost:4222"`
	Username string `env:"NATS_USERNAME"`
	Password string `env:"NATS_PASSWORD"`

	Reconnect struct {
		MaxAttempts int           `env:"NATS_RECONNECT_MAX" envDefault:"-1"` // -1 for unlimited
		Wait        time.Duration `env:"NATS_RECONNECT_WAIT" envDefault:"2s"`
		Buffer      int           `env:"NATS_RECONNECT_BUF_SIZE" envDefault:"8388608"` // 8MB default
	}

	Timeout struct {
		Connect time.Duration `env:"NATS_CONNECT_TIMEOUT" envDefault:"2s"`
		Ping    time.Duration `env:"NATS_PING_INTERVAL" envDefault:"2m"`
	}

	TLS struct {
		Enable bool   `env:"NATS_TLS_ENABLE" envDefault:"false"`
		Cert   string `env:"NATS_TLS_CERT"`
		Key    string `env:"NATS_TLS_KEY"`
		CA     string `env:"NATS_TLS_CA"`
	}
}

func NewNATS() (*NATS, error) {
	cfg := &NATS{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse nats config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid nats config: %w", err)
	}

	return cfg, nil
}

func (c *NATS) Validate() error {
	if err := c.validateConnection(); err != nil {
		return err
	}

	if err := c.validateReconnect(); err != nil {
		return err
	}

	if err := c.validateTimeout(); err != nil {
		return err
	}

	if err := c.validateTLS(); err != nil {
		return err
	}

	if err := c.validateAuth(); err != nil {
		return err
	}

	return nil
}

func (c *NATS) validateConnection() error {
	if c.URL == "" {
		return fmt.Errorf("nats url is required")
	}
	return nil
}

func (c *NATS) validateReconnect() error {
	if c.Reconnect.MaxAttempts < -1 {
		return fmt.Errorf("invalid max reconnect attempts: %d", c.Reconnect.MaxAttempts)
	}

	if c.Reconnect.Wait <= 0 {
		return fmt.Errorf("reconnect wait must be positive")
	}

	if c.Reconnect.Buffer < 0 {
		return fmt.Errorf("reconnect buffer size must be non-negative")
	}

	return nil
}

func (c *NATS) validateTimeout() error {
	if c.Timeout.Connect <= 0 {
		return fmt.Errorf("connect timeout must be positive")
	}

	if c.Timeout.Ping <= 0 {
		return fmt.Errorf("ping interval must be positive")
	}

	return nil
}

func (c *NATS) validateTLS() error {
	if !c.TLS.Enable {
		return nil
	}

	if c.TLS.Cert == "" {
		return fmt.Errorf("tls cert is required when tls is enabled")
	}

	if c.TLS.Key == "" {
		return fmt.Errorf("tls key is required when tls is enabled")
	}

	return nil
}

func (c *NATS) validateAuth() error {
	// Both must be provided together or both empty
	hasUsername := c.Username != ""
	hasPassword := c.Password != ""

	if hasUsername != hasPassword {
		return fmt.Errorf("nats username and password must both be provided or both empty")
	}

	return nil
}
