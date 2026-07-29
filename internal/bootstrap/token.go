package bootstrap

import (
	"github.com/qpubio/qpub-server/internal/shared/token"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

func (a *App) setupToken() error {
	a.token = token.NewService()
	a.logger.Info(log.App, "Token utility initialized")
	return nil
}
