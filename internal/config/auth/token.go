package auth

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Token struct {
	APIKey struct {
		Duration    time.Duration `env:"TOKEN_API_KEY_DURATION" envDefault:"1h"`      // 1 hour
		MaxDuration time.Duration `env:"TOKEN_API_KEY_MAX_DURATION" envDefault:"24h"` // 1 day
	}

	Signing struct {
		Method string `env:"TOKEN_SIGNING_METHOD" envDefault:"HS256"`
	}
}

func NewToken() (*Token, error) {
	cfg := &Token{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse token config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid token config: %w", err)
	}

	return cfg, nil
}

func (c *Token) Validate() error {
	if err := c.validateTokenDuration("api_key", c.APIKey.Duration); err != nil {
		return err
	}

	if !c.isValidSigningMethod(c.Signing.Method) {
		return fmt.Errorf("invalid signing method: %s", c.Signing.Method)
	}

	return nil
}

func (c *Token) validateTokenDuration(tokenType string, duration time.Duration) error {
	if duration <= 0 {
		return fmt.Errorf("%s token duration must be positive", tokenType)
	}

	if duration > 30*24*time.Hour { // 30 days
		return fmt.Errorf("%s token duration cannot exceed 30 days", tokenType)
	}

	return nil
}

func (c *Token) isValidSigningMethod(method string) bool {
	switch method {
	case "HS256", "HS384", "HS512":
		return true
	default:
		return false
	}
}
