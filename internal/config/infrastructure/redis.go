package infrastructure

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Redis struct {
	Host string `env:"REDIS_HOST" envDefault:"localhost"`
	Port string `env:"REDIS_PORT" envDefault:"6379"`

	Options struct {
		DB       int           `env:"REDIS_DB" envDefault:"0"`
		Password string        `env:"REDIS_PASSWORD"`
		Timeout  time.Duration `env:"REDIS_TIMEOUT" envDefault:"5s"`
	}

	Pool struct {
		MinIdle     int           `env:"REDIS_POOL_MIN_IDLE" envDefault:"10"`
		MaxIdle     int           `env:"REDIS_POOL_MAX_IDLE" envDefault:"50"`
		MaxActive   int           `env:"REDIS_POOL_MAX_ACTIVE" envDefault:"0"` // 0 means unlimited
		IdleTimeout time.Duration `env:"REDIS_POOL_IDLE_TIMEOUT" envDefault:"5m"`
	}
}

func NewRedis() (*Redis, error) {
	cfg := &Redis{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse redis config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid redis config: %w", err)
	}

	return cfg, nil
}

func (c *Redis) Validate() error {
	if err := c.validateConnection(); err != nil {
		return err
	}

	if err := c.validateOptions(); err != nil {
		return err
	}

	if err := c.validatePool(); err != nil {
		return err
	}

	return nil
}

func (c *Redis) validateConnection() error {
	if c.Host == "" {
		return fmt.Errorf("redis host is required")
	}

	if c.Port == "" {
		return fmt.Errorf("redis port is required")
	}

	return nil
}

func (c *Redis) validateOptions() error {
	if c.Options.DB < 0 {
		return fmt.Errorf("redis db must be non-negative")
	}

	if c.Options.Timeout <= 0 {
		return fmt.Errorf("redis timeout must be positive")
	}

	return nil
}

func (c *Redis) validatePool() error {
	if c.Pool.MinIdle < 0 {
		return fmt.Errorf("redis min idle connections must be non-negative")
	}

	if c.Pool.MaxIdle < c.Pool.MinIdle {
		return fmt.Errorf("redis max idle connections must be greater than or equal to min idle")
	}

	if c.Pool.MaxActive < 0 {
		return fmt.Errorf("redis max active connections must be non-negative")
	}

	if c.Pool.IdleTimeout <= 0 {
		return fmt.Errorf("redis idle timeout must be positive")
	}

	return nil
}
