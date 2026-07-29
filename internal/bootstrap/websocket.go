package bootstrap

import (
	"github.com/qpubio/qpub-server/internal/api/websocket"
	"github.com/qpubio/qpub-server/internal/bootstrap/container"
	"github.com/qpubio/qpub-server/internal/domain/messaging/connection"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

func (a *App) setupWebSocketServer() error {
	a.logger.Info(log.App, "Setting up WebSocket server...")

	connectionService, err := container.GetTyped[connection.Service](a.container)
	if err != nil {
		return err
	}

	wsServer := websocket.NewServer(a.logger, connectionService)
	a.wsServer = wsServer

	a.cleanup.Register(func() error {
		a.logger.Info(log.App, "Shutting down WebSocket server...")
		a.wsServer.Close()
		return nil
	})

	go wsServer.Run()
	a.logger.Info(log.App, "WebSocket server initialized successfully")
	return nil
}
