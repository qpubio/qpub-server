package infrastructure

import (
	"fmt"
	"path/filepath"

	"github.com/caarlos0/env/v11"
)

type Logger struct {
	Level string `env:"APP_LOG_LEVEL" envDefault:"info"`
	Dir   string `env:"APP_LOG_DIR" envDefault:"logs"`
	JSON  bool   `env:"APP_LOG_JSON" envDefault:"false"`

	Rotation struct {
		MaxSize    int  `env:"APP_LOG_MAX_SIZE" envDefault:"10"`   // megabytes
		MaxBackups int  `env:"APP_LOG_MAX_BACKUPS" envDefault:"3"` // files
		MaxAge     int  `env:"APP_LOG_MAX_AGE" envDefault:"28"`    // days
		Compress   bool `env:"APP_LOG_COMPRESS" envDefault:"true"`
	}
}

func NewLogger() (*Logger, error) {
	cfg := &Logger{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse logger config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid logger config: %w", err)
	}

	// Ensure log directory is absolute path or relative to current directory
	if !filepath.IsAbs(cfg.Dir) {
		cfg.Dir = filepath.Join(".", cfg.Dir)
	}

	return cfg, nil
}

func (c *Logger) Validate() error {
	if !isValidLogLevel(c.Level) {
		return fmt.Errorf("invalid log level: %s", c.Level)
	}

	if c.Rotation.MaxSize <= 0 {
		return fmt.Errorf("max size must be positive")
	}

	if c.Rotation.MaxBackups < 0 {
		return fmt.Errorf("max backups cannot be negative")
	}

	if c.Rotation.MaxAge < 0 {
		return fmt.Errorf("max age cannot be negative")
	}

	return nil
}

func isValidLogLevel(level string) bool {
	switch level {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}
