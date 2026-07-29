package connection

import (
	"github.com/qpubio/qpub-server/internal/domain/messaging/connection"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"sync"
	"time"
)

type repository struct {
	logger               logger.Service
	connections          map[id.ULID]*connection.Connection
	connectionsByProject map[id.Int]map[id.ULID]*connection.Connection
	mu                   sync.RWMutex
}

// NewRepository creates a new connection repository
func NewRepository(logger logger.Service) connection.Repository {
	return &repository{
		logger:               logger,
		connections:          make(map[id.ULID]*connection.Connection),
		connectionsByProject: make(map[id.Int]map[id.ULID]*connection.Connection),
	}
}

// Store adds or updates a connection
func (r *repository) Store(conn *connection.Connection) error {
	if conn == nil {
		return connection.ErrConnectionNotFound
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if this is a new connection
	_, exists := r.connections[conn.ID()]

	// Store in main map
	r.connections[conn.ID()] = conn

	// Store in project map
	if _, exists := r.connectionsByProject[conn.ProjectID()]; !exists {
		r.connectionsByProject[conn.ProjectID()] = make(map[id.ULID]*connection.Connection)
	}
	r.connectionsByProject[conn.ProjectID()][conn.ID()] = conn

	if !exists {
		r.logger.Debug(log.MessagingConnection, `Connection added to repository connectionID=%v projectID=%v`, conn.ID(),
			conn.ProjectID())
	}

	return nil
}

// Remove removes a connection
func (r *repository) Remove(connID id.ULID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get connection
	conn, exists := r.connections[connID]
	if !exists {
		return connection.ErrConnectionNotFound
	}

	// Remove from main map
	delete(r.connections, connID)

	// Remove from project map
	if projectMap, exists := r.connectionsByProject[conn.ProjectID()]; exists {
		delete(projectMap, connID)
		if len(projectMap) == 0 {
			delete(r.connectionsByProject, conn.ProjectID())
		}
	}

	r.logger.Debug(log.MessagingConnection, `Connection removed from repository connectionID=%v projectID=%v`, connID,
		conn.ProjectID())

	return nil
}

// FindByID finds a connection by ID
func (r *repository) FindByID(connID id.ULID) (*connection.Connection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	conn, exists := r.connections[connID]
	if !exists {
		return nil, connection.ErrConnectionNotFound
	}

	return conn, nil
}

// FindAllByProjectID finds all connections for a project
func (r *repository) FindAllByProjectID(projID id.Int) ([]*connection.Connection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projectMap, exists := r.connectionsByProject[projID]
	if !exists {
		return []*connection.Connection{}, nil
	}

	connections := make([]*connection.Connection, 0, len(projectMap))
	for _, conn := range projectMap {
		connections = append(connections, conn)
	}

	return connections, nil
}

// CountByProject returns the number of connections for a specific project
func (r *repository) CountByProject(projID id.Int) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projectMap, exists := r.connectionsByProject[projID]
	if !exists {
		return 0
	}

	return len(projectMap)
}

// CleanStaleConnections removes connections that haven't received a pong in the specified duration
func (r *repository) CleanStaleConnections() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := clock.Now()
	staleTimeout := 2 * time.Minute // No pong received for 2 minutes
	count := 0

	// Find stale connections
	staleConnections := make([]*connection.Connection, 0)
	for _, conn := range r.connections {
		if now.Sub(conn.LastPongAt()) > staleTimeout {
			staleConnections = append(staleConnections, conn)
		}
	}

	// Remove stale connections
	for _, conn := range staleConnections {
		// Remove from main map
		delete(r.connections, conn.ID())

		// Remove from project map
		if projectMap, exists := r.connectionsByProject[conn.ProjectID()]; exists {
			delete(projectMap, conn.ID())
			if len(projectMap) == 0 {
				delete(r.connectionsByProject, conn.ProjectID())
			}
		}

		count++
	}

	if count > 0 {
		r.logger.Info(log.MessagingConnection, `Cleaned stale connections count=%v timeout=%v`, count,
			staleTimeout.String())
	}

	return count
}
