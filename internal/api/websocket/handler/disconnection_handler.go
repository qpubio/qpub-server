package handler

import (
	projectLog "github.com/qpubio/qpub-server/internal/domain/project/log"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/ptr"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

// DisconnectionHandler handles the enhanced disconnection flow with proper stat management
type DisconnectionHandler struct {
	handler *Handler
}

// NewDisconnectionHandler creates a new disconnection handler
func NewDisconnectionHandler(handler *Handler) *DisconnectionHandler {
	return &DisconnectionHandler{
		handler: handler,
	}
}

// HandleDisconnection processes a client disconnection with proper stat management
func (d *DisconnectionHandler) HandleDisconnection(connID id.ULID, clientID id.ULID) {
	h := d.handler

	// Get client details before cleanup
	client, err := h.clientService.GetClient(connID)
	if err != nil {
		h.logger.Error(log.WebSocket, `Client not found during disconnection connectionID=%v clientID=%v error=%v`, connID,
			clientID,
			err)
	}

	var projectID id.Int
	var subscriptionID *id.ULID

	if client != nil {
		projectID = client.ProjectID()
		subscriptionID = client.SubscriptionID()

		// Clean up subscriptions properly using the actual subscription from repository
		if subscriptionID != nil {
			// Get the actual subscription from repository
			sub, err := h.subscriptionService.Get(*subscriptionID)
			if err != nil {
				h.logger.Error(log.WebSocket, `Failed to get subscription for cleanup connectionID=%v clientID=%v subscriptionID=%v error=%v`, connID,
					clientID,
					*subscriptionID,
					err)
			} else {
				// Close subscription properly - this will handle all stat decrements
				if err := h.subscriptionService.CloseSubscription(sub); err != nil {
					h.logger.Error(log.WebSocket, `Failed to close subscription connectionID=%v clientID=%v subscriptionID=%v error=%v`, connID,
						clientID,
						*subscriptionID,
						err)
				}
			}
		}

		// Broadcast client disconnected event
		if h.logBroadcaster != nil {
			conn, err := h.connectionService.Get(connID)
			if err == nil {
				event := projectLog.CreateEvent(projectLog.CreateEventParams{
					Message:      "Client disconnected",
					ConnectionID: ptr.ToPtr(connID),
					ClientID:     ptr.ToPtr(clientID),
					APIKey:       h.getAPIKey(connID),
					RemoteAddr:   conn.RemoteAddr(),
					UserAgent:    ptr.ToPtr(conn.UserAgent()),
					Site:         h.getSite(),
				})
				_ = h.logBroadcaster.PublishLog(projectID, projectLog.EventClientDisconnected, *event)
			}
		}
	}

	// Disconnect client
	if err := h.clientService.Disconnect(connID); err != nil {
		h.logger.Error(log.WebSocket, `Failed to disconnect client connectionID=%v clientID=%v error=%v`, connID,
			clientID,
			err)
	}

	// Close connection (this will decrement connection stats)
	if err := h.connectionService.Close(connID); err != nil {
		h.logger.Error(log.WebSocket, `Failed to close connection connectionID=%v clientID=%v error=%v`, connID,
			clientID,
			err)
	}

	// Broadcast connection closed event
	if h.logBroadcaster != nil && client != nil {
		conn, err := h.connectionService.Get(connID)
		if err == nil {
			event := projectLog.CreateEvent(projectLog.CreateEventParams{
				Message:      "Connection closed",
				ConnectionID: ptr.ToPtr(connID),
				APIKey:       h.getAPIKey(connID),
				RemoteAddr:   conn.RemoteAddr(),
				UserAgent:    ptr.ToPtr(conn.UserAgent()),
				Site:         h.getSite(),
			})
			_ = h.logBroadcaster.PublishLog(projectID, projectLog.EventConnectionClosed, *event)
		}
	}

	// Clean up API key from cache
	h.removeAPIKey(connID)

	// Clean up session state
	h.sessionService.UnregisterConnection(connID, clientID)

	h.logger.Info(log.WebSocket, `Client disconnection completed connectionID=%v clientID=%v projectID=%v hadSubscription=%v`, connID,
		clientID,
		projectID,
		subscriptionID != nil)
}
