package bootstrap

import (
	"fmt"

	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

func (a *App) setupID() error {
	if err := id.Init(&a.config.Shared.ID); err != nil {
		return fmt.Errorf("failed to create id service: %w", err)
	}
	a.logger.Info(log.App, "ID service initialized with HashLength=%d, ULIDLength=%d",
		a.config.Shared.ID.HashLength,
		a.config.Shared.ID.ULIDLength,
	)
	return nil
}
