package subscription

import (
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"sync/atomic"
	"time"
)

// Subscription represents a message subscription
type Subscription struct {
	id         id.ULID
	send       chan []byte
	clientID   id.ULID
	closed     atomic.Bool
	createdAt  time.Time
	lastActive time.Time
}

// New creates a new subscription instance
func New(clientID id.ULID, bufferSize int) *Subscription {
	sub := &Subscription{
		id:         id.NewULID(),
		send:       make(chan []byte, bufferSize),
		clientID:   clientID,
		createdAt:  clock.Now(),
		lastActive: clock.Now(),
	}
	// Initialize as active
	sub.closed.Store(false)
	return sub
}

// ID returns the subscription ID
func (s *Subscription) ID() id.ULID {
	return s.id
}

func (s *Subscription) SetID(subID id.ULID) {
	s.id = subID
}

// Send returns the send channel
func (s *Subscription) SendChan() chan []byte {
	return s.send
}

// ClientID returns the client ID
func (s *Subscription) ClientID() id.ULID {
	return s.clientID
}

// IsClosed returns whether the subscription is closed
func (s *Subscription) IsClosed() bool {
	return s.closed.Load()
}

// IsActive returns whether the subscription is active
func (s *Subscription) IsActive() bool {
	return !s.IsClosed()
}

// CreatedAt returns when the subscription was created
func (s *Subscription) CreatedAt() time.Time {
	return s.createdAt
}

// LastActive returns the time of last activity
func (s *Subscription) LastActive() time.Time {
	return s.lastActive
}

// UpdateActivity updates the last activity timestamp
func (s *Subscription) UpdateActivity() {
	s.lastActive = clock.Now()
}

// Close marks the subscription as closed
func (s *Subscription) Close() {
	if !s.closed.Swap(true) {
		close(s.send)
	}
}
