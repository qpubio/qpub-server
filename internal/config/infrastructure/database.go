package infrastructure

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Database struct {
	Driver string `env:"DB_DRIVER" envDefault:"postgres"` // postgres, cockroach, mysql, etc.

	Host     string `env:"DB_HOST"`
	Port     string `env:"DB_PORT" envDefault:"5432"`
	Name     string `env:"DB_NAME"`
	User     string `env:"DB_USER"`
	Password string `env:"DB_PASSWORD"`

	Options struct {
		SSLMode     string        `env:"DB_SSLMODE" envDefault:"disable"`
		SSLRootCert string        `env:"DB_SSLROOTCERT"` // Path to CA certificate
		SSLCert     string        `env:"DB_SSLCERT"`     // Path to client certificate
		SSLKey      string        `env:"DB_SSLKEY"`      // Path to client key
		Timezone    string        `env:"DB_TIMEZONE" envDefault:"UTC"`
		Timeout     time.Duration `env:"DB_TIMEOUT" envDefault:"30s"`
	}

	// Driver-specific configurations
	Postgres struct {
		SearchPath string `env:"DB_POSTGRES_SEARCH_PATH"`
	}

	Cockroach struct {
		Retries int `env:"DB_COCKROACH_RETRIES" envDefault:"3"`
	}
}

func NewDatabase() (*Database, error) {
	cfg := &Database{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid database config: %w", err)
	}

	return cfg, nil
}

func (c *Database) Validate() error {
	if err := c.validateRequired(); err != nil {
		return err
	}

	if err := c.validateOptions(); err != nil {
		return err
	}

	return nil
}

func (c *Database) validateRequired() error {
	if c.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Port == "" {
		return fmt.Errorf("database port is required")
	}

	if c.Name == "" {
		return fmt.Errorf("database name is required")
	}

	if c.User == "" {
		return fmt.Errorf("database user is required")
	}

	// Password is required for PostgreSQL
	// For CockroachDB, password is required for secure mode, optional for insecure mode
	if c.Password == "" && c.Driver == "postgres" {
		return fmt.Errorf("database password is required")
	}

	if c.Password == "" && c.Driver == "cockroach" && c.Options.SSLMode != "disable" {
		return fmt.Errorf("database password is required for CockroachDB in secure mode")
	}

	// Note: CockroachDB in insecure mode (sslmode=disable) accepts connections
	// regardless of password, but we allow setting a password for tools like Adminer

	return nil
}

func (c *Database) validateOptions() error {
	if !c.isValidSSLMode(c.Options.SSLMode) {
		return fmt.Errorf("invalid ssl mode: %s", c.Options.SSLMode)
	}

	if _, err := time.LoadLocation(c.Options.Timezone); err != nil {
		return fmt.Errorf("invalid timezone: %s", c.Options.Timezone)
	}

	return nil
}

func (c *Database) isValidSSLMode(mode string) bool {
	validModes := map[string]bool{
		"disable":     true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}
	return validModes[mode]
}
