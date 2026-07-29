package channel

import (
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"time"
)

// Channel represents a messaging channel
type Channel struct {
	id                 id.ULID
	name               Name
	instanceID         id.ULID // ID of the server instance that manages this channel
	localSubscriptions int
	isActive           bool // Whether this instance is subscribed to the broker
	createdAt          time.Time
	lastActivity       time.Time
}

// New creates a new channel
func New(name Name, instanceID id.ULID) *Channel {
	now := clock.Now()
	return &Channel{
		id:                 id.NewULID(),
		name:               name,
		instanceID:         instanceID,
		localSubscriptions: 0,
		isActive:           false,
		createdAt:          now,
		lastActivity:       now,
	}
}

// ID returns the channel ID
func (c *Channel) ID() id.ULID {
	return c.id
}

// Name returns the channel name
func (c *Channel) Name() Name {
	return c.name
}

// RawName returns the client-facing channel name
func (c *Channel) RawName() string {
	return c.name.Raw()
}

// FullName returns the fully qualified channel name with project prefix
func (c *Channel) FullName() string {
	return c.name.Full()
}

// InstanceID returns the instance ID
func (c *Channel) InstanceID() id.ULID {
	return c.instanceID
}

// ProjectID returns the project ID
func (c *Channel) ProjectID() id.Int {
	return c.name.ProjectID()
}

// LocalSubscriptionCount returns the number of local subscriptions
func (c *Channel) LocalSubscriptionCount() int {
	return c.localSubscriptions
}

// IncrementLocalSubscriptions adds a subscription to the channel
func (c *Channel) IncrementLocalSubscriptions() {
	c.localSubscriptions++
	c.UpdateActivity()
}

// DecrementLocalSubscriptions removes a subscriber from the channel
func (c *Channel) DecrementLocalSubscriptions() {
	if c.HasLocalSubscribers() {
		c.localSubscriptions--
		c.UpdateActivity()
	}
}

// IsActive returns whether the channel is active
func (c *Channel) IsActive() bool {
	return c.isActive
}

// SetActive sets the active status of the channel
func (c *Channel) SetActive(active bool) {
	c.isActive = active
	c.UpdateActivity()
}

// CreatedAt returns the creation time
func (c *Channel) CreatedAt() time.Time {
	return c.createdAt
}

// LastActivity returns the last activity time
func (c *Channel) LastActivity() time.Time {
	return c.lastActivity
}

// UpdateLastActivity updates the last activity time
func (c *Channel) UpdateActivity() {
	c.lastActivity = clock.Now()
}

func (c *Channel) HasLocalSubscribers() bool {
	return c.localSubscriptions > 0
}
