package bootstrap

import (
	"github.com/qpubio/qpub-server/internal/shared/permission"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

func (a *App) setupPermission() error {
	a.permission = permission.NewService()
	a.logger.Info(log.App, "Permission service initialized")
	return nil
}
