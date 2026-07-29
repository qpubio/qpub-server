package bootstrap

import (
	"fmt"
	"os"

	"github.com/qpubio/qpub-server/internal/config"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"github.com/joho/godotenv"
)

func (a *App) setupConfig() error {
	if _, err := os.Stat(".env"); err == nil {
		_ = godotenv.Load()
	} else {
		fmt.Println("Warning: No .env file found, using system environment variables")
	}

	cfg, err := config.NewConfig()
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}
	a.config = cfg

	a.registerPostLoggerInit(func() {
		a.logger.Info(log.Config, "Cluster Config: %+v", cfg.Infrastructure.Cluster)
	})
	return nil
}

func (a *App) registerPostLoggerInit(fn func()) {
	a.postLoggerInit = append(a.postLoggerInit, fn)
}
