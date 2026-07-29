package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	sessionApp "github.com/qpubio/qpub-server/internal/application/service/messaging/session"
	"github.com/qpubio/qpub-server/internal/config"
	clientDomain "github.com/qpubio/qpub-server/internal/domain/messaging/client"
	"github.com/qpubio/qpub-server/internal/domain/messaging/connection"
	"github.com/qpubio/qpub-server/internal/domain/messaging/protocol"
	"github.com/qpubio/qpub-server/internal/domain/messaging/publication"
	"github.com/qpubio/qpub-server/internal/domain/messaging/subscription"
	projectLog "github.com/qpubio/qpub-server/internal/domain/project/log"
	logBroadcast "github.com/qpubio/qpub-server/internal/domain/project/log/broadcast"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/permission"
	"github.com/qpubio/qpub-server/internal/shared/ptr"
	"github.com/qpubio/qpub-server/internal/shared/type/log"

	"github.com/gorilla/websocket"
)

// Handler handles WebSocket connections and messages
type Handler struct {
	config              *config.Config
	logger              logger.Service
	instanceID          id.ULID
	permissionService   permission.Service
	connectionService   connection.Service
	clientService       clientDomain.Service
	subscriptionService subscription.Service
	sessionService      *sessionApp.Service
	publicationService  publication.Service
	logBroadcaster      logBroadcast.Service
	upgrader            websocket.Upgrader
	apiKeyCache         map[id.ULID]string // connectionID -> apiPublicKey
	apiKeyCacheMu       sync.RWMutex
	pingIDCounter       atomic.Int64 // auto-incrementing counter for ping message IDs
}

// NewHandler creates a new WebSocket handler
func NewHandler(
	config *config.Config,
	logger logger.Service,
	instanceID id.ULID,
	permissionService permission.Service,
	connectionService connection.Service,
	clientService clientDomain.Service,
	subscriptionService subscription.Service,
	sessionService *sessionApp.Service,
	publicationService publication.Service,
	logBroadcaster logBroadcast.Service,
) *Handler {
	return &Handler{
		config:              config,
		logger:              logger,
		instanceID:          instanceID,
		permissionService:   permissionService,
		connectionService:   connectionService,
		clientService:       clientService,
		subscriptionService: subscriptionService,
		sessionService:      sessionService,
		publicationService:  publicationService,
		logBroadcaster:      logBroadcaster,
		apiKeyCache:         make(map[id.ULID]string),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now
				// TODO: Implement proper origin checking
				return true
			},
		},
	}
}

// getSite returns the site identifier in the format "{region}-{zone}"
func (h *Handler) getSite() string {
	return h.config.Infrastructure.Cluster.Site()
}

// getAPIKey returns the API key public key for a given connection ID
func (h *Handler) getAPIKey(connID id.ULID) *string {
	h.apiKeyCacheMu.RLock()
	defer h.apiKeyCacheMu.RUnlock()
	if apiKey, ok := h.apiKeyCache[connID]; ok {
		return &apiKey
	}
	return nil
}

// removeAPIKey removes the API key from cache when connection closes
func (h *Handler) removeAPIKey(connID id.ULID) {
	h.apiKeyCacheMu.Lock()
	defer h.apiKeyCacheMu.Unlock()
	delete(h.apiKeyCache, connID)
}

// HandleConnection upgrades an HTTP connection to WebSocket and handles it
func (h *Handler) HandleConnection(
	w http.ResponseWriter,
	r *http.Request,
	projectID *id.Int,
	apiKeyID *id.Int,
	apiPublicKey *string,
	alias *string,
	permission *json.RawMessage,
) {
	// Upgrade connection to WebSocket
	wsConn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error(log.WebSocket, `WebSocket upgrade failed error=%v projectID=%v remoteAddr=%v`, err,
			projectID,
			r.RemoteAddr)
		return
	}

	// Create domain connection object
	conn := connection.New(*projectID, r.RemoteAddr, r.UserAgent())

	// Configure WebSocket connection
	wsConn.SetReadLimit(1024 * 1024) // 1MB max message size
	wsConn.SetPongHandler(func(string) error {
		wsConn.SetReadDeadline(clock.Now().Add(60 * time.Second))
		conn.UpdatePong()
		return nil
	})

	// Create the send handler function that will be registered with the connection service
	sendHandler := func(msg []byte) error {
		return wsConn.WriteMessage(websocket.TextMessage, msg)
	}

	// Register connection with service
	err = h.connectionService.Register(conn, sendHandler)
	if err != nil {
		h.logger.Error(log.WebSocket, `Failed to register connection error=%v projectID=%v`, err,
			projectID)
		wsConn.Close()
		return
	}

	// Store API key for this connection
	h.apiKeyCacheMu.Lock()
	h.apiKeyCache[conn.ID()] = *apiPublicKey
	h.apiKeyCacheMu.Unlock()

	// Broadcast connection opened event with a small delay
	// This allows the client to subscribe to _logs before receiving the event
	if h.logBroadcaster != nil {
		userAgent := r.UserAgent()
		origin := r.Header.Get("Origin")
		event := projectLog.CreateEvent(projectLog.CreateEventParams{
			Message:      "Connection opened",
			ConnectionID: ptr.ToPtr(conn.ID()),
			APIKey:       apiPublicKey,
			RemoteAddr:   r.RemoteAddr,
			UserAgent:    ptr.ToPtr(userAgent),
			Origin:       ptr.ToPtr(origin),
			Site:         h.getSite(),
		})
		go func() {
			// Small delay to allow client to subscribe to _logs channel first
			time.Sleep(150 * time.Millisecond)
			_ = h.logBroadcaster.PublishLog(*projectID, projectLog.EventConnectionOpened, *event)
		}()
	}

	// Create client for this connection
	client, err := h.clientService.Connect(conn.ID(), *projectID, *apiKeyID, alias, permission)
	if err != nil {
		h.logger.Error(log.WebSocket, `Failed to create client error=%v connectionID=%v projectID=%v`, err,
			conn.ID(),
			projectID)
		h.connectionService.Close(conn.ID())
		wsConn.Close()
		return
	}

	// Broadcast client connected event with a small delay
	// This allows the client to subscribe to _logs before receiving the event
	if h.logBroadcaster != nil {
		userAgent := r.UserAgent()
		origin := r.Header.Get("Origin")
		event := projectLog.CreateEvent(projectLog.CreateEventParams{
			Message:      "Client connected",
			ConnectionID: ptr.ToPtr(conn.ID()),
			ClientID:     ptr.ToPtr(client.ID()),
			APIKey:       apiPublicKey,
			RemoteAddr:   r.RemoteAddr,
			UserAgent:    ptr.ToPtr(userAgent),
			Origin:       ptr.ToPtr(origin),
			Site:         h.getSite(),
		})
		go func() {
			// Small delay to allow client to subscribe to _logs channel first
			time.Sleep(200 * time.Millisecond)
			_ = h.logBroadcaster.PublishLog(*projectID, projectLog.EventClientConnected, *event)
		}()
	}

	// Send initial connection message to client
	h.sendConnectedMessage(conn.ID(), client.ID())

	h.sessionService.RegisterConnection(conn.ID(), *projectID, client.ID())

	h.logger.Info(log.WebSocket, `WebSocket connection established connectionID=%v clientID=%v projectID=%v remoteAddr=%v`, conn.ID(),
		client.ID(),
		projectID,
		r.RemoteAddr)

	// Start reading messages
	go h.readWebSocketMessages(wsConn, conn.ID(), client.ID())

	// Start ping routine
	go h.pingWebSocket(wsConn, conn.ID())
}

// readWebSocketMessages reads messages from the WebSocket
func (h *Handler) readWebSocketMessages(wsConn *websocket.Conn, connID id.ULID, clientID id.ULID) {
	defer func() {
		// This will be called when the connection is closed
		h.handleDisconnection(connID, clientID)
		wsConn.Close()
	}()

	// Set initial read deadline
	wsConn.SetReadDeadline(clock.Now().Add(60 * time.Second))

	for {
		// Read message
		_, message, err := wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error(log.WebSocket, `WebSocket read error error=%v connectionID=%v clientID=%v`, err,
					connID,
					clientID)

				// Broadcast connection error event
				conn, connErr := h.connectionService.Get(connID)
				if connErr == nil && h.logBroadcaster != nil {
					event := projectLog.CreateEvent(projectLog.CreateEventParams{
						Message:      "WebSocket read error",
						ConnectionID: ptr.ToPtr(connID),
						ClientID:     ptr.ToPtr(clientID),
						APIKey:       h.getAPIKey(connID),
						RemoteAddr:   conn.RemoteAddr(),
						UserAgent:    ptr.ToPtr(conn.UserAgent()),
						Site:         h.getSite(),
						Error: &projectLog.ErrorDetails{
							Message: err.Error(),
						},
					})

					_ = h.logBroadcaster.PublishLog(conn.ProjectID(), projectLog.EventConnectionError, *event)
				}
			} else {
				h.logger.Info(log.WebSocket, `WebSocket connection closed connectionID=%v clientID=%v`, connID,
					clientID)
			}
			break
		}

		// Update connection stats
		conn, err := h.connectionService.Get(connID)
		if err == nil {
			conn.UpdateRecv(len(message))
		}

		// Handle message
		go h.handleMessage(connID, clientID, message)
	}
}

// pingWebSocket sends periodic pings to keep the connection alive
func (h *Handler) pingWebSocket(wsConn *websocket.Conn, connID id.ULID) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := wsConn.WriteControl(
			websocket.PingMessage,
			[]byte{},
			clock.Now().Add(10*time.Second),
		); err != nil {
			h.logger.Error(log.WebSocket, `Error sending ping, closing connection error=%v connectionID=%v`, err,
				connID)
			h.connectionService.Close(connID)
			return
		}
	}
}

// handleDisconnection processes a client disconnection
func (h *Handler) handleDisconnection(connID id.ULID, clientID id.ULID) {
	// Use the enhanced disconnection handler for proper stat management
	disconnectionHandler := NewDisconnectionHandler(h)
	disconnectionHandler.HandleDisconnection(connID, clientID)
}

// sendConnectedMessage sends a connected message to the client
func (h *Handler) sendConnectedMessage(connID id.ULID, clientID id.ULID) {
	// Get server ID using cluster helper
	serverID := h.config.Infrastructure.Cluster.ServerID(h.instanceID)

	// Get client-provided alias if available
	client, _ := h.clientService.GetClient(connID)
	var alias *string
	if client != nil && client.Alias() != nil {
		alias = client.Alias()
	}

	// Create connection message
	connMsg := protocol.NewConnectionMessage(
		protocol.ActionConnected,
		connID,
		&clientID,
		&serverID,
		alias,
		nil, // No error
	)

	// Marshal and send
	messageBytes, err := json.Marshal(connMsg)
	if err != nil {
		h.logger.Error(log.WebSocket, `Failed to marshal connected message error=%v connectionID=%v clientID=%v`, err,
			connID,
			clientID)
		return
	}

	err = h.clientService.SendMessage(connID, messageBytes)
	if err != nil {
		h.logger.Error(log.WebSocket, `Failed to send connected message error=%v connectionID=%v clientID=%v`, err,
			connID,
			clientID)
	}
}

// handleMessage processes incoming WebSocket messages
func (h *Handler) handleMessage(connID id.ULID, clientID id.ULID, message []byte) {
	// Parse the base message to determine action
	var baseMsg protocol.Message
	if err := json.Unmarshal(message, &baseMsg); err != nil {
		h.logger.Error(log.WebSocket, `Failed to parse message error=%v connectionID=%v clientID=%v`, err,
			connID,
			clientID)

		h.sendErrorResponse(
			connID,
			protocol.ErrInvalidMessage,
			protocol.HrefInvalidMessage,
			"Invalid message format",
			protocol.StatusCodeBadRequest,
		)
		return
	}

	// Process message based on action
	switch baseMsg.Action {
	case protocol.ActionSubscribe:
		h.handleSubscribeMessage(connID, clientID, message)
	case protocol.ActionUnsubscribe:
		h.handleUnsubscribeMessage(connID, clientID, message)
	case protocol.ActionPublish:
		h.handlePublishMessage(connID, clientID, message)
	case protocol.ActionPing:
		h.handlePingMessage(connID, clientID, message)
	case protocol.ActionPong:
		h.handlePongMessage(connID, clientID, message)
	default:
		h.logger.Warn(log.WebSocket, `Unknown message action action=%v connectionID=%v clientID=%v`, baseMsg.Action,
			connID,
			clientID)

		h.sendErrorResponse(
			connID,
			protocol.ErrInvalidAction,
			protocol.HrefInvalidAction,
			fmt.Sprintf("Unsupported action: %d", baseMsg.Action),
			protocol.StatusCodeBadRequest,
		)
	}
}

// sendErrorResponse sends an error response to the client
func (h *Handler) sendErrorResponse(connID id.ULID, code protocol.Code, href protocol.Href, message string, statusCode protocol.StatusCode) {
	errorInfo := &protocol.ErrorInfo{
		Code:       int(code),
		Href:       string(href),
		Message:    message,
		StatusCode: int(statusCode),
	}

	errorMsg := &protocol.Message{
		Action: protocol.ActionError,
		Error:  errorInfo,
	}

	// Marshal and send the error message
	msgBytes, err := json.Marshal(errorMsg)
	if err != nil {
		h.logger.Error(log.WebSocket, `Failed to marshal error message error=%v connectionID=%v`, err,
			connID)
		return
	}

	if err := h.clientService.SendMessage(connID, msgBytes); err != nil {
		h.logger.Error(log.WebSocket, `Failed to send error message error=%v connectionID=%v`, err,
			connID)
	}
}

// handleSubscribeMessage processes a subscribe message
func (h *Handler) handleSubscribeMessage(connID id.ULID, clientID id.ULID, message []byte) {
	// Parse the channel message
	var channelMsg protocol.ChannelMessage
	if err := json.Unmarshal(message, &channelMsg); err != nil {
		h.logger.Error(log.WebSocket, `Failed to parse subscribe message error=%v connectionID=%v clientID=%v`, err,
			connID,
			clientID)

		h.sendErrorResponse(
			connID,
			protocol.ErrInvalidMessage,
			protocol.HrefInvalidMessage,
			"Invalid subscribe message format",
			protocol.StatusCodeBadRequest,
		)
		return
	}

	// Get client to access project ID and permissions
	client, err := h.clientService.GetClient(connID)
	if err != nil {
		h.logger.Error(log.WebSocket, `Client not found for subscription error=%v connectionID=%v clientID=%v`, err,
			connID,
			clientID)

		h.sendErrorResponse(
			connID,
			protocol.ErrNotFound,
			protocol.HrefNotFound,
			"Client not found",
			protocol.StatusCodeNotFound,
		)
		return
	}

	// Check permissions for subscribe operation
	permData := []byte(*client.Permission())
	canSubscribe, err := h.permissionService.CanSubscribe(permData, channelMsg.Channel)
	if err != nil {
		h.logger.Error(log.WebSocket, `Error checking subscription permissions error=%v connectionID=%v clientID=%v channel=%v`, err,
			connID,
			clientID,
			channelMsg.Channel)

		h.sendErrorResponse(
			connID,
			protocol.ErrBadRequest,
			protocol.HrefBadRequest,
			"Invalid permission format",
			protocol.StatusCodeBadRequest,
		)
		return
	}

	if !canSubscribe {
		h.logger.Warn(log.WebSocket, `Subscription permission denied connectionID=%v clientID=%v channel=%v`, connID,
			clientID,
			channelMsg.Channel)

		h.sendErrorResponse(
			connID,
			protocol.ErrForbidden,
			protocol.HrefForbidden,
			fmt.Sprintf("Permission denied to subscribe to channel: %s", channelMsg.Channel),
			protocol.StatusCodeForbidden,
		)
		return
	}

	// Create or get subscription via session service
	sub := h.sessionService.GetOrCreateSubscription(clientID)
	if client.SubscriptionID() == nil {
		client.SetSubscriptionID(sub.ID())
	}

	h.logger.Debug(log.WebSocket, `Using session subscription clientID=%v subscriptionID=%v channel=%v`, clientID,
		sub.ID(),
		channelMsg.Channel)

	// Subscribe to channel
	err = h.subscriptionService.Subscribe(channelMsg.Channel, sub, client.ProjectID())
	if err != nil {
		h.logger.Error(log.WebSocket, `Failed to subscribe to channel error=%v connectionID=%v clientID=%v channel=%v`, err,
			connID,
			clientID,
			channelMsg.Channel)

		// Subscription service will send appropriate state message
		return
	}

	// Broadcast subscription created event
	if h.logBroadcaster != nil {
		conn, err := h.connectionService.Get(connID)
		if err == nil {
			event := projectLog.CreateEvent(projectLog.CreateEventParams{
				Message:      "Subscription created",
				ConnectionID: ptr.ToPtr(connID),
				ClientID:     ptr.ToPtr(clientID),
				ChannelName:  ptr.ToPtr(channelMsg.Channel),
				APIKey:       h.getAPIKey(connID),
				RemoteAddr:   conn.RemoteAddr(),
				UserAgent:    ptr.ToPtr(conn.UserAgent()),
				Site:         h.getSite(),
			})
			go func() {
				time.Sleep(250 * time.Millisecond)
				_ = h.logBroadcaster.PublishLog(client.ProjectID(), projectLog.EventSubscribed, *event)
			}()
		}
	}

	h.logger.Info(log.WebSocket, `Client subscribed to channel connectionID=%v clientID=%v subscriptionID=%v channel=%v`, connID,
		clientID,
		sub.ID(),
		channelMsg.Channel)
}

// handleUnsubscribeMessage processes an unsubscribe message
func (h *Handler) handleUnsubscribeMessage(connID id.ULID, clientID id.ULID, message []byte) {
	// Parse the channel message
	var channelMsg protocol.ChannelMessage
	if err := json.Unmarshal(message, &channelMsg); err != nil {
		h.logger.Error(log.WebSocket, `Failed to parse unsubscribe message error=%v connectionID=%v clientID=%v`, err,
			connID,
			clientID)

		h.sendErrorResponse(
			connID,
			protocol.ErrInvalidMessage,
			protocol.HrefInvalidMessage,
			"Invalid unsubscribe message format",
			protocol.StatusCodeBadRequest,
		)
		return
	}

	// Get client to access subscription ID and project ID
	client, err := h.clientService.GetClient(connID)
	if err != nil {
		h.logger.Error(log.WebSocket, `Client not found for unsubscription error=%v connectionID=%v clientID=%v`, err,
			connID,
			clientID)

		h.sendErrorResponse(
			connID,
			protocol.ErrNotFound,
			protocol.HrefNotFound,
			"Client not found",
			protocol.StatusCodeNotFound,
		)
		return
	}

	// Check if client has a subscription
	if client.SubscriptionID() == nil {
		h.logger.Warn(log.WebSocket, `Client tried to unsubscribe without a subscription connectionID=%v clientID=%v channel=%v`, connID,
			clientID,
			channelMsg.Channel)

		h.sendErrorResponse(
			connID,
			protocol.ErrSubscriptionClosed,
			protocol.HrefSubscriptionClosed,
			"No active subscription",
			protocol.StatusCodeNotFound,
		)
		return
	}

	// Get subscription
	sub, err := h.subscriptionService.Get(*client.SubscriptionID())
	if err != nil {
		h.logger.Error(log.WebSocket, `Subscription not found for unsubscription error=%v connectionID=%v clientID=%v subscriptionID=%v channel=%v`, err,
			connID,
			clientID,
			*client.SubscriptionID(),
			channelMsg.Channel)

		h.sendErrorResponse(
			connID,
			protocol.ErrSubscriptionClosed,
			protocol.HrefSubscriptionClosed,
			"Subscription not found",
			protocol.StatusCodeNotFound,
		)
		return
	}

	// Unsubscribe from channel
	err = h.subscriptionService.Unsubscribe(channelMsg.Channel, sub, client.ProjectID())
	if err != nil {
		h.logger.Error(log.WebSocket, `Failed to unsubscribe from channel error=%v connectionID=%v clientID=%v subscriptionID=%v channel=%v`, err,
			connID,
			clientID,
			*client.SubscriptionID(),
			channelMsg.Channel)

		// Subscription service will send appropriate state message
		return
	}

	// Broadcast subscription closed event
	if h.logBroadcaster != nil {
		conn, err := h.connectionService.Get(connID)
		if err == nil {
			event := projectLog.CreateEvent(projectLog.CreateEventParams{
				Message:      "Subscription closed",
				ConnectionID: ptr.ToPtr(connID),
				ClientID:     ptr.ToPtr(clientID),
				ChannelName:  ptr.ToPtr(channelMsg.Channel),
				APIKey:       h.getAPIKey(connID),
				RemoteAddr:   conn.RemoteAddr(),
				UserAgent:    ptr.ToPtr(conn.UserAgent()),
				Site:         h.getSite(),
			})
			_ = h.logBroadcaster.PublishLog(client.ProjectID(), projectLog.EventUnsubscribed, *event)
		}
	}

	h.logger.Info(log.WebSocket, `Client unsubscribed from channel connectionID=%v clientID=%v subscriptionID=%v channel=%v`, connID,
		clientID,
		*client.SubscriptionID(),
		channelMsg.Channel)
}

// handlePublishMessage processes a publish message
func (h *Handler) handlePublishMessage(connID id.ULID, clientID id.ULID, message []byte) {
	// Parse the data message
	var dataMsg protocol.DataMessage
	if err := json.Unmarshal(message, &dataMsg); err != nil {
		h.logger.Error(log.WebSocket, `Failed to parse publish message error=%v connectionID=%v clientID=%v`, err,
			connID,
			clientID)

		h.sendErrorResponse(
			connID,
			protocol.ErrInvalidMessage,
			protocol.HrefInvalidMessage,
			"Invalid publish message format",
			protocol.StatusCodeBadRequest,
		)
		return
	}

	// Get client to check authentication and get project ID
	client, err := h.clientService.GetClient(connID)
	if err != nil {
		h.logger.Error(log.WebSocket, `Client not found for publish error=%v connectionID=%v clientID=%v`, err,
			connID,
			clientID)

		h.sendErrorResponse(
			connID,
			protocol.ErrNotFound,
			protocol.HrefNotFound,
			"Client not found",
			protocol.StatusCodeNotFound,
		)
		return
	}

	// Check permissions for publish operation
	permData := []byte(*client.Permission())
	canPublish, err := h.permissionService.CanPublish(permData, dataMsg.Channel)
	if err != nil {
		h.logger.Error(log.WebSocket, `Error checking publish permissions error=%v connectionID=%v clientID=%v channel=%v`, err,
			connID,
			clientID,
			dataMsg.Channel)

		// Use appropriate error code for permission parsing issues
		h.sendErrorResponse(
			connID,
			protocol.ErrBadRequest,
			protocol.HrefBadRequest,
			"Invalid permission format",
			protocol.StatusCodeBadRequest,
		)
		return
	}

	if !canPublish {
		h.logger.Warn(log.WebSocket, `Publish permission denied connectionID=%v clientID=%v channel=%v`, connID,
			clientID,
			dataMsg.Channel)

		// Use forbidden error code for permission denied
		h.sendErrorResponse(
			connID,
			protocol.ErrForbidden,
			protocol.HrefForbidden,
			fmt.Sprintf("Permission denied to publish to channel: %s", dataMsg.Channel),
			protocol.StatusCodeForbidden,
		)
		return
	}

	// Convert protocol messages to domain messages
	payloads := make([]publication.Payload, 0)
	for _, msg := range dataMsg.Messages {
		payloads = append(payloads, publication.Payload{
			Alias: msg.Alias,
			Event: msg.Event,
			Data:  msg.Data,
		})
	}

	// Create publication message
	pubMsg, err := publication.Create(publication.CreateParams{
		ProjectID:   client.ProjectID(),
		ChannelName: dataMsg.Channel,
		Messages:    payloads,
	})
	if err != nil {
		h.logger.Error(log.WebSocket, `Failed to create publication message error=%v connectionID=%v clientID=%v`, err,
			connID,
			clientID)

		h.sendErrorResponse(
			connID,
			protocol.ErrInternal,
			protocol.HrefInternal,
			"Failed to create publication message",
			protocol.StatusCodeInternal,
		)
		return
	}

	// Publish message
	_, err = h.publicationService.Publish(connID, pubMsg, false)
	if err != nil {
		h.logger.Error(log.WebSocket, `Failed to publish message error=%v connectionID=%v clientID=%v channel=%v`, err,
			connID,
			clientID,
			dataMsg.Channel)

		// Publication service will send appropriate state message
		return
	}

	h.logger.Info(log.WebSocket, `Client published message connectionID=%v clientID=%v channel=%v messageCount=%v`, connID,
		clientID,
		dataMsg.Channel,
		len(dataMsg.Messages))
}

// handlePingMessage processes a client ping message and responds with pong
func (h *Handler) handlePingMessage(connID id.ULID, clientID id.ULID, message []byte) {
	// Parse the ping message
	var pingMsg protocol.PingMessage
	if err := json.Unmarshal(message, &pingMsg); err != nil {
		h.logger.Error(log.WebSocket, `Failed to parse ping message error=%v connectionID=%v clientID=%v`, err,
			connID,
			clientID)

		h.sendErrorResponse(
			connID,
			protocol.ErrInvalidMessage,
			protocol.HrefInvalidMessage,
			"Invalid ping message format",
			protocol.StatusCodeBadRequest,
		)
		return
	}

	// Update connection stats for client ping
	conn, err := h.connectionService.Get(connID)
	if err == nil {
		conn.UpdateClientPing()
	}

	// Create and send pong response with the original ping timestamp
	pongMsg := protocol.NewPongMessage(pingMsg.ID)
	pongBytes, err := json.Marshal(pongMsg)
	if err != nil {
		h.logger.Error(log.WebSocket, `Failed to marshal pong message error=%v connectionID=%v clientID=%v`, err,
			connID,
			clientID)
		return
	}

	// Send pong response to client
	if err := h.clientService.SendMessage(connID, pongBytes); err != nil {
		h.logger.Error(log.WebSocket, `Failed to send pong message error=%v connectionID=%v clientID=%v`, err,
			connID,
			clientID)
		return
	}

	h.logger.Debug(log.WebSocket, `Responded to client ping with pong connectionID=%v clientID=%v pingID=%v`, connID,
		clientID,
		pingMsg.ID)
}

// handlePongMessage processes a client pong message (response to server ping)
func (h *Handler) handlePongMessage(connID id.ULID, clientID id.ULID, message []byte) {
	// Parse the pong message
	var pongMsg protocol.PongMessage
	if err := json.Unmarshal(message, &pongMsg); err != nil {
		h.logger.Error(log.WebSocket, `Failed to parse pong message error=%v connectionID=%v clientID=%v`, err,
			connID,
			clientID)

		h.sendErrorResponse(
			connID,
			protocol.ErrInvalidMessage,
			protocol.HrefInvalidMessage,
			"Invalid pong message format",
			protocol.StatusCodeBadRequest,
		)
		return
	}

	// Update connection stats for client pong
	conn, err := h.connectionService.Get(connID)
	if err == nil {
		conn.UpdateClientPong()
	}

	h.logger.Debug(log.WebSocket, `Received client pong message connectionID=%v clientID=%v pongID=%v`, connID,
		clientID,
		pongMsg.ID)
}

// SendPingToClient sends an application-level ping message to a client
func (h *Handler) SendPingToClient(connID id.ULID) error {
	// Create ping message with auto-incrementing ID
	pingID := int(h.pingIDCounter.Add(1))
	pingMsg := protocol.NewPingMessage(pingID)
	pingBytes, err := json.Marshal(pingMsg)
	if err != nil {
		h.logger.Error(log.WebSocket, `Failed to marshal ping message error=%v connectionID=%v`, err,
			connID)
		return err
	}

	// Send ping message to client
	if err := h.clientService.SendMessage(connID, pingBytes); err != nil {
		h.logger.Error(log.WebSocket, `Failed to send ping message to client error=%v connectionID=%v`, err,
			connID)
		return err
	}

	h.logger.Debug(log.WebSocket, `Sent application-level ping to client connectionID=%v pingID=%v`, connID,
		pingMsg.ID)

	return nil
}
