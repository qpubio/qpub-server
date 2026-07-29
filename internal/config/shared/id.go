package shared

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type ID struct {
	HashSalt   string `env:"HASHID_SALT"`
	HashLength int    `env:"HASHID_LENGTH" envDefault:"11"`
	ULIDLength int    `env:"ULID_LENGTH" envDefault:"22"`
}

func NewID() (*ID, error) {
	cfg := &ID{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse ID config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ID config: %w", err)
	}

	return cfg, nil
}

func (c *ID) Validate() error {
	if c.HashSalt == "" {
		return fmt.Errorf("HASHID_SALT is required")
	}

	if !isValidLength(c.HashLength) {
		return fmt.Errorf("invalid hash length: %d", c.HashLength)
	}

	if !isValidLength(c.ULIDLength) {
		return fmt.Errorf("invalid ULID length: %d", c.ULIDLength)
	}

	return nil
}

func isValidLength(length int) bool {
	return length > 0 && length <= 100 // reasonable upper limit
}
