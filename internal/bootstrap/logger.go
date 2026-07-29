package bootstrap

import (
	"fmt"

	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

func (a *App) setupLogger() error {
	loggerService, err := logger.New(&a.config.Infrastructure.Logger)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	a.logger = loggerService

	a.cleanup.Register(func() error {
		a.logger.Info(log.App, "Shutting down logger...")
		return nil
	})

	for _, fn := range a.postLoggerInit {
		fn()
	}

	a.logger.Info(log.App, "Logger initialized with JSON=%v, Dir=%s",
		a.config.Infrastructure.Logger.JSON,
		a.config.Infrastructure.Logger.Dir,
	)
	return nil
}
