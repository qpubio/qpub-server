package connection

import (
	"github.com/qpubio/qpub-server/internal/domain/messaging/connection"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"sync"
)

// Service implements the connection service
type Service struct {
	logger       logger.Service
	instanceID   id.ULID
	repository   connection.Repository
	sendHandlers map[id.ULID]func([]byte) error
	mutex        sync.RWMutex
	writeMutexes map[id.ULID]*sync.Mutex
}

// NewService creates a new connection service
func NewService(
	logger logger.Service,
	instanceID id.ULID,
	repository connection.Repository,
) connection.Service {
	return &Service{
		logger:       logger,
		instanceID:   instanceID,
		repository:   repository,
		sendHandlers: make(map[id.ULID]func([]byte) error),
		writeMutexes: make(map[id.ULID]*sync.Mutex),
	}
}

// Register registers a new connection with a send handler
func (s *Service) Register(conn *connection.Connection, sendHandler func([]byte) error) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Store connection in repository
	if err := s.repository.Store(conn); err != nil {
		return err
	}

	// Store send handler
	s.sendHandlers[conn.ID()] = sendHandler
	// Create write mutex for this connection
	s.writeMutexes[conn.ID()] = &sync.Mutex{}
	// Set connection state to open
	conn.SetState(connection.StateOpen)
	s.repository.Store(conn)

	s.logger.Info(log.MessagingConnection, `Connection registered connectionID=%v projectID=%v remoteAddr=%v`, conn.ID(),
		conn.ProjectID(),
		conn.RemoteAddr())

	// Note: Connection stats are tracked via snapshot service (every 500ms)
	// which counts actual connections in the repository

	return nil
}

// Unregister removes a connection
func (s *Service) Unregister(connID id.ULID) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Get connection from repository first for logging
	conn, err := s.repository.FindByID(connID)
	if err != nil {
		return err
	}

	// Remove send handler and mutex
	delete(s.sendHandlers, connID)
	delete(s.writeMutexes, connID)

	// Remove connection from repository (actually remove it, don't just mark as closed)
	if err := s.repository.Remove(connID); err != nil {
		s.logger.Warn(log.MessagingConnection, `Failed to remove connection from repository connectionID=%v error=%v`, connID,
			err)
		return err
	}

	s.logger.Info(log.MessagingConnection, `Connection unregistered connectionID=%v projectID=%v`, connID,
		conn.ProjectID())

	return nil
}

// Send sends a message to a connection
func (s *Service) Send(connID id.ULID, message []byte) error {
	s.mutex.RLock()
	sendHandler, sendHandlerExists := s.sendHandlers[connID]
	writeMutex, writeMutexExists := s.writeMutexes[connID]
	s.mutex.RUnlock()

	if !sendHandlerExists || !writeMutexExists {
		return connection.ErrConnectionNotFound
	}

	// Get connection from repository to update stats
	conn, err := s.repository.FindByID(connID)
	if err != nil {
		return err
	}

	// Check if connection is open
	if conn.IsClosed() {
		return connection.ErrConnectionClosed
	}

	// Lock this connection's write mutex before sending
	writeMutex.Lock()
	defer writeMutex.Unlock()

	// Send message using handler
	if err := sendHandler(message); err != nil {
		s.logger.Error(log.MessagingConnection, `Failed to send message connectionID=%v error=%v`, connID,
			err)
		return connection.ErrSendFailed
	}

	// Update connection-level stats (for tracking purposes)
	conn.UpdateSent(len(message))
	s.repository.Store(conn)

	return nil
}

// Broadcast sends a message to all connections for a project
func (s *Service) Broadcast(projID id.Int, message []byte) error {
	// Get all connections for project
	conns, err := s.repository.FindAllByProjectID(projID)
	if err != nil {
		return err
	}

	s.logger.Info(log.MessagingConnection, `Broadcasting message to connections projectID=%v connectionCount=%v messageSize=%v`, projID,
		len(conns),
		len(message))

	errorCount := 0
	for _, conn := range conns {
		if err := s.Send(conn.ID(), message); err != nil {
			errorCount++
			// Continue with other connections
		}
	}

	if errorCount > 0 {
		s.logger.Warn(log.MessagingConnection, `Failed to broadcast to some connections projectID=%v errorCount=%v totalConnections=%v`, projID,
			errorCount,
			len(conns))
	}

	return nil
}

// Close marks a connection as closed and removes it
func (s *Service) Close(connID id.ULID) error {
	// Get connection from repository
	conn, err := s.repository.FindByID(connID)
	if err != nil {
		return err
	}

	// Mark as closed before unregistering (for proper state tracking)
	if conn.State() == connection.StateOpen {
		conn.SetState(connection.StateClosed)
		s.repository.Store(conn)
	}

	// Unregister connection (removes from repository)
	// Note: Connection stats are tracked via snapshot service which counts actual connections
	return s.Unregister(connID)
}

// CloseAllByProject closes all connections for a project
func (s *Service) CloseAllByProject(projID id.Int) error {
	// Get all connections for project
	conns, err := s.repository.FindAllByProjectID(projID)
	if err != nil {
		return err
	}

	s.logger.Info(log.MessagingConnection, `Closing all connections for project projectID=%v connectionCount=%v`, projID,
		len(conns))

	for _, conn := range conns {
		// Close each connection
		go s.Close(conn.ID()) // Use goroutine to not block
	}

	return nil
}

// CleanStaleConnections removes stale connections
func (s *Service) CleanStaleConnections() (int, error) {
	// Use repository to clean stale connections
	count := s.repository.CleanStaleConnections()

	if count > 0 {
		s.logger.Info(log.MessagingConnection, `Cleaned stale connections count=%v`, count)
	}

	return count, nil
}

// Get returns a connection by ID
func (s *Service) Get(connID id.ULID) (*connection.Connection, error) {
	return s.repository.FindByID(connID)
}
