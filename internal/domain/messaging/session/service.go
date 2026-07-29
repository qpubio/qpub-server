package session

import (
	"context"
	"encoding/json"

	"github.com/qpubio/qpub-server/internal/domain/messaging/client"
	"github.com/qpubio/qpub-server/internal/domain/messaging/subscription"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Service manages the lifecycle of a messaging session bound to one connection.
type Service interface {
	// Open creates or restores a session for a connection.
	Open(ctx context.Context, params OpenParams) (*Session, error)

	// Close tears down the session and all subscriptions.
	Close(ctx context.Context, connectionID id.ULID) error

	// Get returns the session for a connection.
	Get(connectionID id.ULID) (*Session, error)

	// Subscribe adds a channel to the session subscription set.
	Subscribe(ctx context.Context, connectionID id.ULID, channelRawName string) (*subscription.Subscription, error)

	// Unsubscribe removes a channel from the session subscription set.
	Unsubscribe(ctx context.Context, connectionID id.ULID, channelRawName string) error

	// Subscription returns the reusable subscription for the session.
	Subscription(connectionID id.ULID) (*subscription.Subscription, error)
}

// OpenParams holds inputs for opening a session.
type OpenParams struct {
	ConnectionID id.ULID
	ProjectID    id.Int
	APIKeyID     id.Int
	Alias        *string
	Permission   *json.RawMessage
}

// Session binds transport, client identity, and subscription state.
type Session struct {
	connectionID   id.ULID
	projectID      id.Int
	client         *client.Client
	subscription   *subscription.Subscription
	subscriptionID id.ULID
}

// NewSession creates a session aggregate from its parts.
func NewSession(
	connectionID id.ULID,
	projectID id.Int,
	cl *client.Client,
	sub *subscription.Subscription,
) *Session {
	var subID id.ULID
	if sub != nil {
		subID = sub.ID()
	}
	return &Session{
		connectionID:   connectionID,
		projectID:      projectID,
		client:         cl,
		subscription:   sub,
		subscriptionID: subID,
	}
}

func (s *Session) ConnectionID() id.ULID { return s.connectionID }
func (s *Session) ProjectID() id.Int     { return s.projectID }
func (s *Session) Client() *client.Client {
	return s.client
}
func (s *Session) Subscription() *subscription.Subscription {
	return s.subscription
}
func (s *Session) SubscriptionID() id.ULID {
	return s.subscriptionID
}

// IsActive reports whether the session has an open subscription.
func (s *Session) IsActive() bool {
	return s != nil && s.subscription != nil && s.subscription.IsActive()
}
