package client

import (
	clientDomain "github.com/qpubio/qpub-server/internal/domain/messaging/client"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"sync"
	"time"
)

type repository struct {
	logger                  logger.Service
	clientsByID             map[id.ULID]*clientDomain.Client
	clientsByConnectionID   map[id.ULID]*clientDomain.Client
	clientsBySubscriptionID map[id.ULID]*clientDomain.Client
	clientsByAlias          map[string]map[id.Int]*clientDomain.Client // map[alias]map[projectID]client
	clientsByProjectID      map[id.Int]map[id.ULID]*clientDomain.Client
	mu                      sync.RWMutex
}

// NewRepository creates a new client repository
func NewRepository(logger logger.Service) clientDomain.Repository {
	return &repository{
		logger:                  logger,
		clientsByID:             make(map[id.ULID]*clientDomain.Client),
		clientsByConnectionID:   make(map[id.ULID]*clientDomain.Client),
		clientsBySubscriptionID: make(map[id.ULID]*clientDomain.Client),
		clientsByAlias:          make(map[string]map[id.Int]*clientDomain.Client),
		clientsByProjectID:      make(map[id.Int]map[id.ULID]*clientDomain.Client),
	}
}

// Create adds a new client
func (r *repository) Create(c *clientDomain.Client) error {
	if c == nil {
		return clientDomain.ErrClientNotFound
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if client already exists
	if _, exists := r.clientsByID[c.ID()]; exists {
		return clientDomain.ErrClientAlreadyExists
	}

	// Store in main map
	r.clientsByID[c.ID()] = c

	// Store in connection ID map
	r.clientsByConnectionID[c.ConnectionID()] = c

	// Store in project ID map
	if _, exists := r.clientsByProjectID[c.ProjectID()]; !exists {
		r.clientsByProjectID[c.ProjectID()] = make(map[id.ULID]*clientDomain.Client)
	}
	r.clientsByProjectID[c.ProjectID()][c.ID()] = c

	// Store in subscription ID map if set
	if c.SubscriptionID() != nil {
		r.clientsBySubscriptionID[*c.SubscriptionID()] = c
	}

	// Store in client alias map if set
	if c.Alias() != nil {
		alias := *c.Alias()
		if _, exists := r.clientsByAlias[alias]; !exists {
			r.clientsByAlias[alias] = make(map[id.Int]*clientDomain.Client)
		}
		r.clientsByAlias[alias][c.ProjectID()] = c
	}

	r.logger.Debug(log.MessagingClient, `Client created in repository clientID=%v connectionID=%v projectID=%v`, c.ID(),
		c.ConnectionID(),
		c.ProjectID())

	return nil
}

// Update updates an existing client
func (r *repository) Update(c *clientDomain.Client) error {
	if c == nil {
		return clientDomain.ErrClientNotFound
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if client exists
	existingClient, exists := r.clientsByID[c.ID()]
	if !exists {
		return clientDomain.ErrClientNotFound
	}

	// Remove from connection ID map if changed
	if existingClient.ConnectionID() != c.ConnectionID() {
		delete(r.clientsByConnectionID, existingClient.ConnectionID())
		r.clientsByConnectionID[c.ConnectionID()] = c
	}

	// Update subscription ID if changed
	if existingClient.SubscriptionID() != nil &&
		(c.SubscriptionID() == nil || *existingClient.SubscriptionID() != *c.SubscriptionID()) {
		// Remove old subscription ID
		delete(r.clientsBySubscriptionID, *existingClient.SubscriptionID())
	}
	if c.SubscriptionID() != nil {
		// Add new subscription ID
		r.clientsBySubscriptionID[*c.SubscriptionID()] = c
	}

	// Update client ID if changed
	if existingClient.Alias() != nil &&
		(c.Alias() == nil || *existingClient.Alias() != *c.Alias()) {
		// Remove old client alias
		oldAlias := *existingClient.Alias()
		if projectMap, exists := r.clientsByAlias[oldAlias]; exists {
			delete(projectMap, existingClient.ProjectID())
			if len(projectMap) == 0 {
				delete(r.clientsByAlias, oldAlias)
			}
		}
	}
	if c.Alias() != nil {
		// Add new client alias
		newAlias := *c.Alias()
		if _, exists := r.clientsByAlias[newAlias]; !exists {
			r.clientsByAlias[newAlias] = make(map[id.Int]*clientDomain.Client)
		}
		r.clientsByAlias[newAlias][c.ProjectID()] = c
	}

	// Update in main map
	r.clientsByID[c.ID()] = c

	r.logger.Debug(log.MessagingClient, `Client updated in repository clientID=%v connectionID=%v state=%v`, c.ID(),
		c.ConnectionID(),
		c.State())

	return nil
}

// Delete removes a client
func (r *repository) Delete(c *clientDomain.Client) error {
	if c == nil {
		return clientDomain.ErrClientNotFound
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if client exists
	if _, exists := r.clientsByID[c.ID()]; !exists {
		return clientDomain.ErrClientNotFound
	}

	// Remove from all maps
	delete(r.clientsByID, c.ID())
	delete(r.clientsByConnectionID, c.ConnectionID())

	// Remove from project ID map
	if projectMap, exists := r.clientsByProjectID[c.ProjectID()]; exists {
		delete(projectMap, c.ID())
		if len(projectMap) == 0 {
			delete(r.clientsByProjectID, c.ProjectID())
		}
	}

	// Remove from subscription ID map if set
	if c.SubscriptionID() != nil {
		delete(r.clientsBySubscriptionID, *c.SubscriptionID())
	}

	// Remove from client alias map if set
	if c.Alias() != nil {
		alias := *c.Alias()
		if projectMap, exists := r.clientsByAlias[alias]; exists {
			delete(projectMap, c.ProjectID())
			if len(projectMap) == 0 {
				delete(r.clientsByAlias, alias)
			}
		}
	}

	r.logger.Debug(log.MessagingClient, `Client deleted from repository clientID=%v connectionID=%v`, c.ID(),
		c.ConnectionID())

	return nil
}

// FindByID finds a client by ID
func (r *repository) FindByID(clientID id.ULID) (*clientDomain.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	client, exists := r.clientsByID[clientID]
	if !exists {
		return nil, clientDomain.ErrClientNotFound
	}

	return client, nil
}

// FindByConnectionID finds a client by connection ID
func (r *repository) FindByConnectionID(connID id.ULID) (*clientDomain.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	client, exists := r.clientsByConnectionID[connID]
	if !exists {
		return nil, clientDomain.ErrClientNotFound
	}

	return client, nil
}

// FindBySubscriptionID finds a client by subscription ID
func (r *repository) FindBySubscriptionID(subID id.ULID) (*clientDomain.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	client, exists := r.clientsBySubscriptionID[subID]
	if !exists {
		return nil, clientDomain.ErrClientNotFound
	}

	return client, nil
}

// FindByAlias finds a client by client-provided alias
func (r *repository) FindByAlias(alias string, projID id.Int) (*clientDomain.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projectMap, exists := r.clientsByAlias[alias]
	if !exists {
		return nil, clientDomain.ErrClientNotFound
	}

	client, exists := projectMap[projID]
	if !exists {
		return nil, clientDomain.ErrClientNotFound
	}

	return client, nil
}

// FindAllByProjectID finds all clients for a project
func (r *repository) FindAllByProjectID(projID id.Int) ([]*clientDomain.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projectMap, exists := r.clientsByProjectID[projID]
	if !exists {
		return []*clientDomain.Client{}, nil
	}

	clients := make([]*clientDomain.Client, 0, len(projectMap))
	for _, client := range projectMap {
		clients = append(clients, client)
	}

	return clients, nil
}

// CleanDisconnectedClients removes disconnected clients older than the specified duration
func (r *repository) CleanDisconnectedClients() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := clock.Now()
	disconnectedTimeout := 30 * time.Minute
	count := 0

	// Find disconnected clients to remove
	clientsToRemove := make([]*clientDomain.Client, 0)
	for _, c := range r.clientsByID {
		if c.State() == clientDomain.StateDisconnected && now.Sub(c.LastActiveAt()) > disconnectedTimeout {
			clientsToRemove = append(clientsToRemove, c)
		}
	}

	// Remove disconnected clients
	for _, c := range clientsToRemove {
		// Remove from all maps
		delete(r.clientsByID, c.ID())
		delete(r.clientsByConnectionID, c.ConnectionID())

		// Remove from project ID map
		if projectMap, exists := r.clientsByProjectID[c.ProjectID()]; exists {
			delete(projectMap, c.ID())
			if len(projectMap) == 0 {
				delete(r.clientsByProjectID, c.ProjectID())
			}
		}

		// Remove from subscription ID map if set
		if c.SubscriptionID() != nil {
			delete(r.clientsBySubscriptionID, *c.SubscriptionID())
		}

		// Remove from client alias map if set
		if c.Alias() != nil {
			alias := *c.Alias()
			if projectMap, exists := r.clientsByAlias[alias]; exists {
				delete(projectMap, c.ProjectID())
				if len(projectMap) == 0 {
					delete(r.clientsByAlias, alias)
				}
			}
		}

		count++
	}

	if count > 0 {
		r.logger.Info(log.MessagingClient, `Cleaned disconnected clients count=%v timeout=%v`, count,
			disconnectedTimeout.String())
	}

	return count
}
