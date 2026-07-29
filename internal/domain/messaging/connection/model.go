package connection

import (
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"sync/atomic"
	"time"
)

// State represents the connection state
type State string

const (
	// Connection states
	StateOpening State = "opening"
	StateOpen    State = "open"
	StateClosing State = "closing"
	StateClosed  State = "closed"
	StateError   State = "error"
)

// Connection represents a WebSocket connection
type Connection struct {
	id               id.ULID
	projectID        id.Int
	alias            *string
	remoteAddr       string
	userAgent        string
	state            atomic.Value
	createdAt        time.Time
	lastPingAt       time.Time
	lastPongAt       time.Time
	lastClientPingAt time.Time
	lastClientPongAt time.Time
	lastSentAt       time.Time
	lastRecvAt       time.Time
	messagesSent     atomic.Int64
	messagesRecv     atomic.Int64
	bytesSent        atomic.Int64
	bytesRecv        atomic.Int64
}

// New creates a new connection
func New(projID id.Int, remoteAddr, userAgent string) *Connection {
	conn := &Connection{
		id:               id.NewULID(),
		projectID:        projID,
		remoteAddr:       remoteAddr,
		userAgent:        userAgent,
		createdAt:        clock.Now(),
		lastPingAt:       clock.Now(),
		lastPongAt:       clock.Now(),
		lastClientPingAt: clock.Now(),
		lastClientPongAt: clock.Now(),
		lastSentAt:       clock.Now(),
		lastRecvAt:       clock.Now(),
	}
	conn.state.Store(StateOpening)
	conn.messagesSent.Store(0)
	conn.messagesRecv.Store(0)
	conn.bytesSent.Store(0)
	conn.bytesRecv.Store(0)
	return conn
}

// ID returns the connection ID
func (c *Connection) ID() id.ULID {
	return c.id
}

// ProjectID returns the project ID
func (c *Connection) ProjectID() id.Int {
	return c.projectID
}

// Alias returns the client-provided alias if set
func (c *Connection) Alias() *string {
	return c.alias
}

// SetAlias sets the client-provided alias
func (c *Connection) SetAlias(alias string) {
	c.alias = &alias
}

// RemoteAddr returns the remote address
func (c *Connection) RemoteAddr() string {
	return c.remoteAddr
}

// UserAgent returns the user agent
func (c *Connection) UserAgent() string {
	return c.userAgent
}

// State returns the current connection state
func (c *Connection) State() State {
	return c.state.Load().(State)
}

// SetState updates the connection state
func (c *Connection) SetState(state State) {
	c.state.Store(state)
}

// IsOpen returns whether the connection is open
func (c *Connection) IsOpen() bool {
	return c.State() == StateOpen
}

// IsClosed returns whether the connection is closed
func (c *Connection) IsClosed() bool {
	state := c.State()
	return state == StateClosed || state == StateError
}

// CreatedAt returns the creation time
func (c *Connection) CreatedAt() time.Time {
	return c.createdAt
}

// LastPingAt returns the last ping time
func (c *Connection) LastPingAt() time.Time {
	return c.lastPingAt
}

// UpdatePing updates the last ping time
func (c *Connection) UpdatePing() {
	c.lastPingAt = clock.Now()
}

// LastPongAt returns the last pong time
func (c *Connection) LastPongAt() time.Time {
	return c.lastPongAt
}

// UpdatePong updates the last pong time
func (c *Connection) UpdatePong() {
	c.lastPongAt = clock.Now()
}

// LastClientPingAt returns the last client ping time
func (c *Connection) LastClientPingAt() time.Time {
	return c.lastClientPingAt
}

// UpdateClientPing updates the last client ping time
func (c *Connection) UpdateClientPing() {
	c.lastClientPingAt = clock.Now()
}

// LastClientPongAt returns the last client pong time
func (c *Connection) LastClientPongAt() time.Time {
	return c.lastClientPongAt
}

// UpdateClientPong updates the last client pong time
func (c *Connection) UpdateClientPong() {
	c.lastClientPongAt = clock.Now()
}

// LastSentAt returns the last message sent time
func (c *Connection) LastSentAt() time.Time {
	return c.lastSentAt
}

// LastRecvAt returns the last message received time
func (c *Connection) LastRecvAt() time.Time {
	return c.lastRecvAt
}

// UpdateSent updates the last message sent time and counters
func (c *Connection) UpdateSent(bytes int) {
	c.lastSentAt = clock.Now()
	c.messagesSent.Add(1)
	c.bytesSent.Add(int64(bytes))
}

// UpdateRecv updates the last message received time and counters
func (c *Connection) UpdateRecv(bytes int) {
	c.lastRecvAt = clock.Now()
	c.messagesRecv.Add(1)
	c.bytesRecv.Add(int64(bytes))
}

// MessagesSent returns the number of messages sent
func (c *Connection) MessagesSent() int64 {
	return c.messagesSent.Load()
}

// MessagesRecv returns the number of messages received
func (c *Connection) MessagesRecv() int64 {
	return c.messagesRecv.Load()
}

// BytesSent returns the number of bytes sent
func (c *Connection) BytesSent() int64 {
	return c.bytesSent.Load()
}

// BytesRecv returns the number of bytes received
func (c *Connection) BytesRecv() int64 {
	return c.bytesRecv.Load()
}
