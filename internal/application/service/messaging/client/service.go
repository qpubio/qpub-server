package client

import (
	"encoding/json"
	clientDomain "github.com/qpubio/qpub-server/internal/domain/messaging/client"
	"github.com/qpubio/qpub-server/internal/domain/messaging/connection"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

type Service struct {
	logger            logger.Service
	instanceID        id.ULID
	repository        clientDomain.Repository
	connectionService connection.Service
}

func NewService(
	logger logger.Service,
	instanceID id.ULID,
	repository clientDomain.Repository,
	connectionService connection.Service,
) clientDomain.Service {
	return &Service{
		logger:            logger,
		instanceID:        instanceID,
		repository:        repository,
		connectionService: connectionService,
	}
}

// Connect registers a new client connection
func (s *Service) Connect(
	connID id.ULID,
	projID id.Int,
	apiKeyID id.Int,
	clientID *string,
	permission *json.RawMessage,
) (*clientDomain.Client, error) {
	// Check if a client with this connection ID already exists
	existingClient, err := s.repository.FindByConnectionID(connID)
	if err == nil && existingClient != nil {
		s.logger.Warn(log.MessagingClient, `Client with connection ID already exists connectionID=%v clientID=%v projectID=%v`, connID,
			existingClient.ID(),
			projID)
		return existingClient, nil
	}

	// Create new client
	newClient := clientDomain.New(connID, projID, apiKeyID, clientID, permission)

	// Set state to connected
	newClient.SetState(clientDomain.StateConnected)

	// Save client
	if err := s.repository.Create(newClient); err != nil {
		s.logger.Error(log.MessagingClient, `Failed to create client connectionID=%v projectID=%v error=%v`, connID,
			projID,
			err)
		return nil, err
	}

	s.logger.Info(log.MessagingClient, `Client connected clientID=%v connectionID=%v projectID=%v`, newClient.ID(),
		connID,
		projID)

	return newClient, nil
}

// Disconnect marks a client as disconnected
func (s *Service) Disconnect(connID id.ULID) error {
	clnt, err := s.repository.FindByConnectionID(connID)
	if err != nil {
		s.logger.Error(log.MessagingClient, `Client not found for disconnection connectionID=%v error=%v`, connID,
			err)
		return err
	}

	// Update client state
	clnt.SetState(clientDomain.StateDisconnected)

	// Update client in repository
	if err := s.repository.Update(clnt); err != nil {
		s.logger.Error(log.MessagingClient, `Failed to update client state on disconnect clientID=%v connectionID=%v error=%v`, clnt.ID(),
			connID,
			err)
		return err
	}

	s.logger.Info(log.MessagingClient, `Client disconnected clientID=%v connectionID=%v projectID=%v`, clnt.ID(),
		connID,
		clnt.ProjectID())

	return nil
}

// GetClient retrieves a client by connection ID
func (s *Service) GetClient(connID id.ULID) (*clientDomain.Client, error) {
	return s.repository.FindByConnectionID(connID)
}

// SendMessage sends a message to a client
func (s *Service) SendMessage(connID id.ULID, message []byte) error {
	// Find client to ensure it exists and is connected
	c, err := s.repository.FindByConnectionID(connID)
	if err != nil {
		return clientDomain.ErrClientNotFound
	}

	if !c.IsConnected() {
		return clientDomain.ErrClientDisconnected
	}

	// Update client activity
	c.UpdateActivity()
	s.repository.Update(c)

	// Send message via connection service
	return s.connectionService.Send(connID, message)
}

// BroadcastToProject sends a message to all clients in a project
func (s *Service) BroadcastToProject(projID id.Int, message []byte) error {
	clients, err := s.repository.FindAllByProjectID(projID)
	if err != nil {
		s.logger.Error(log.MessagingClient, `Failed to find clients for broadcast projectID=%v error=%v`, projID,
			err)
		return err
	}

	s.logger.Info(log.MessagingClient, `Broadcasting message to clients projectID=%v clientCount=%v messageSize=%v`, projID,
		len(clients),
		len(message))

	errorCount := 0
	// Send to all connected clients
	for _, c := range clients {
		if c.IsConnected() {
			if err := s.SendMessage(c.ConnectionID(), message); err != nil {
				errorCount++
				s.logger.Error(log.MessagingClient, `Failed to send message to client clientID=%v connectionID=%v error=%v`, c.ID(),
					c.ConnectionID(),
					err)
				// Continue with other clients
			}
		}
	}

	if errorCount > 0 {
		s.logger.Warn(log.MessagingClient, `Failed to send message to some clients projectID=%v errorCount=%v totalClients=%v`, projID,
			errorCount,
			len(clients))
	}

	return nil
}

// CleanDisconnectedClients removes disconnected clients
func (s *Service) CleanDisconnectedClients() (int, error) {
	count := s.repository.CleanDisconnectedClients()

	if count > 0 {
		s.logger.Info(log.MessagingClient, `Cleaned disconnected clients count=%v`, count)
	}

	return count, nil
}
