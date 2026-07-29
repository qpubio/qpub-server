package bootstrap

import (
	"fmt"

	"github.com/qpubio/qpub-server/internal/infrastructure/nats"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

func setupNATS(app *App) error {
	natsService, err := nats.New(&app.config.Infrastructure.NATS)
	if err != nil {
		return fmt.Errorf("failed to create nats service: %w", err)
	}
	if err := natsService.Connect(); err != nil {
		return fmt.Errorf("failed to connect to nats: %w", err)
	}
	app.nats = natsService
	app.cleanup.Register(func() error {
		app.logger.Info(log.NATS, "Closing NATS connection...")
		return natsService.Close()
	})
	authStatus := "without authentication"
	if app.config.Infrastructure.NATS.Username != "" {
		authStatus = "with authentication"
	}
	tlsStatus := ""
	if app.config.Infrastructure.NATS.TLS.Enable {
		tlsStatus = " (TLS enabled)"
	}
	app.logger.Info(log.NATS, "NATS initialized successfully at %s %s%s",
		app.config.Infrastructure.NATS.URL, authStatus, tlsStatus)
	return nil
}
