package bootstrap

import (
	"fmt"
	"time"

	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

func (a *App) setupTimezone() error {
	a.logger.Info(log.App, "Setting up timezone...")
	if err := a.config.Infrastructure.Time.Validate(); err != nil {
		return fmt.Errorf("timezone validation failed: %w", err)
	}
	time.Local = time.UTC
	clock.Init()
	a.logger.Info(log.App, "Timezone set to UTC")
	a.logger.Info(log.App, "Current time: %s", clock.Now().Format(time.RFC3339))
	return nil
}
