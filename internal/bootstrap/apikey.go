package bootstrap

import (
	"github.com/qpubio/qpub-server/internal/shared/apikey"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

func (a *App) setupAPIKeyParser() error {
	a.apikeyParser = apikey.NewParser()
	a.logger.Info(log.App, "APIKey parser initialized")
	return nil
}
