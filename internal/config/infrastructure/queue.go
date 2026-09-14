package infrastructure

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Queue struct {
	Driver string `env:"QUEUE_DRIVER" envDefault:"jetstream"`

	Worker struct {
		Concurrency int           `env:"QUEUE_CONCURRENCY" envDefault:"10"`
		PollInterval time.Duration `env:"QUEUE_POLL_INTERVAL" envDefault:"1s"`
	}

	Defaults struct {
		VisibilityTimeout time.Duration `env:"QUEUE_VISIBILITY_TIMEOUT" envDefault:"30s"`
		MaxAttempts       int           `env:"QUEUE_RETRY_LIMIT" envDefault:"25"`
		Retention         time.Duration `env:"QUEUE_RETENTION" envDefault:"168h"`
	}

	Cleanup struct {
		BatchSize            int           `env:"QUEUE_CLEANUP_BATCH_SIZE" envDefault:"1000"`
		WorkerStaleDisplay   time.Duration `env:"QUEUE_WORKER_STALE_DISPLAY" envDefault:"90s"`
		WorkerStaleDelete    time.Duration `env:"QUEUE_WORKER_STALE_DELETE" envDefault:"168h"`
		JobSuccessRetention  time.Duration `env:"QUEUE_JOB_SUCCESS_RETENTION" envDefault:"168h"`
		JobFailureRetention  time.Duration `env:"QUEUE_JOB_FAILURE_RETENTION" envDefault:"2160h"`
	}
}

func NewQueue() (*Queue, error) {
	cfg := &Queue{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse queue config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid queue config: %w", err)
	}

	return cfg, nil
}

func (c *Queue) Validate() error {
	if c.Worker.Concurrency < 1 {
		return fmt.Errorf("concurrency must be at least 1")
	}
	if c.Defaults.MaxAttempts < 0 {
		return fmt.Errorf("retry limit cannot be negative")
	}
	if c.Defaults.VisibilityTimeout <= 0 {
		return fmt.Errorf("visibility timeout must be positive")
	}
	if c.Driver != "jetstream" {
		return fmt.Errorf("unsupported queue driver: %s", c.Driver)
	}
	return nil
}
