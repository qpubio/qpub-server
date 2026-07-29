package client

import (
	"encoding/json"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"sync/atomic"
	"time"
)

// State represents the client state
type State string

const (
	// Client states
	StateConnecting   State = "connecting"
	StateConnected    State = "connected"
	StateDisconnected State = "disconnected"
)

// Client represents a connected client
type Client struct {
	id             id.ULID
	connectionID   id.ULID
	projectID      id.Int
	apiKeyID       id.Int
	alias          *string
	permission     *json.RawMessage
	state          atomic.Value // State
	authenticated  atomic.Bool
	createdAt      time.Time
	lastActiveAt   time.Time
	subscriptionID *id.ULID // Optional linked subscription
}

// New creates a new client instance
func New(
	connID id.ULID,
	projID id.Int,
	apiKeyID id.Int,
	alias *string,
	permission *json.RawMessage,
) *Client {
	client := &Client{
		id:           id.NewULID(),
		connectionID: connID,
		projectID:    projID,
		apiKeyID:     apiKeyID,
		alias:        alias,
		permission:   permission,
		createdAt:    clock.Now(),
		lastActiveAt: clock.Now(),
	}
	client.state.Store(StateConnecting)
	client.authenticated.Store(false)
	return client
}

// ID returns the client ID
func (c *Client) ID() id.ULID {
	return c.id
}

// ConnectionID returns the associated connection ID
func (c *Client) ConnectionID() id.ULID {
	return c.connectionID
}

// ProjectID returns the project ID
func (c *Client) ProjectID() id.Int {
	return c.projectID
}

// APIKeyID returns the API key ID
func (c *Client) APIKeyID() id.Int {
	return c.apiKeyID
}

// SetAPIKeyID sets the API key ID
func (c *Client) SetAPIKeyID(apiKeyID id.Int) {
	c.apiKeyID = apiKeyID
}

// Alias returns the client-provided identifier
func (c *Client) Alias() *string {
	return c.alias
}

// Permission returns the client permission
func (c *Client) Permission() *json.RawMessage {
	return c.permission
}

// SetAlias sets the client-provided identifier
func (c *Client) SetAlias(alias string) {
	c.alias = &alias
}

// State returns the current client state
func (c *Client) State() State {
	return c.state.Load().(State)
}

// SetState updates the client state
func (c *Client) SetState(state State) {
	c.state.Store(state)
	c.UpdateActivity()
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.State() == StateConnected
}

// IsAuthenticated returns whether the client is authenticated
func (c *Client) IsAuthenticated() bool {
	return c.authenticated.Load()
}

// SetAuthenticated sets the authenticated state
func (c *Client) SetAuthenticated(authenticated bool) {
	c.authenticated.Store(authenticated)
}

// CreatedAt returns the creation time
func (c *Client) CreatedAt() time.Time {
	return c.createdAt
}

// LastActiveAt returns the last activity time
func (c *Client) LastActiveAt() time.Time {
	return c.lastActiveAt
}

// UpdateActivity updates the last activity timestamp
func (c *Client) UpdateActivity() {
	c.lastActiveAt = clock.Now()
}

// SubscriptionID returns the associated subscription ID if any
func (c *Client) SubscriptionID() *id.ULID {
	return c.subscriptionID
}

// SetSubscriptionID sets the associated subscription ID
func (c *Client) SetSubscriptionID(subID id.ULID) {
	c.subscriptionID = &subID
}

// RemoveSubscriptionID removes the associated subscription ID
func (c *Client) RemoveSubscriptionID() {
	c.subscriptionID = nil
}
