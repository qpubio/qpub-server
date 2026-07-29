package bootstrap

import (
	"github.com/qpubio/qpub-server/internal/shared/period"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

func (a *App) setupPeriod() error {
	a.period = period.NewService(a.config.App.Features.TestMode)
	a.logger.Info(log.App, "Period service initialized")
	return nil
}
