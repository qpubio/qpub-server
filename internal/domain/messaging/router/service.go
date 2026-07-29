package router

import (
	"context"
	"encoding/json"

	"github.com/qpubio/qpub-server/internal/domain/messaging/envelope"
	"github.com/qpubio/qpub-server/internal/domain/messaging/publication"
	"github.com/qpubio/qpub-server/internal/domain/messaging/receipt"
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// PublishRequest carries an inbound publish from any transport adapter.
type PublishRequest struct {
	ConnectionID  id.ULID
	ProjectID     id.Int
	Channel       string
	Message       *publication.Message
	SkipTelemetry bool
	Source        envelope.Source
}

// Service routes inbound publishes and broker deliveries through the runtime.
type Service interface {
	// PublishInbound accepts, validates, and routes an inbound publish.
	// Returns an ingress receipt and publish metadata on success.
	PublishInbound(ctx context.Context, req PublishRequest) (*receipt.Receipt, *publication.PublishResult, error)

	// OnBrokerMessage handles a message received from the distributed broker.
	OnBrokerMessage(ctx context.Context, env *envelope.Envelope) error

	// DeliverOutbound enqueues an outbound envelope for a subscriber connection.
	DeliverOutbound(ctx context.Context, env *envelope.Envelope, subscriptionID id.ULID, connectionID id.ULID) (*receipt.Receipt, error)

	// EnsureChannelListening registers a NATS broker listener for a channel when needed.
	EnsureChannelListening(fullChannelName string) error
}

// PermissionChecker validates publish and subscribe permissions for a session.
type PermissionChecker interface {
	CanPublish(connID id.ULID, channel string, permission *json.RawMessage) error
	CanSubscribe(connID id.ULID, channel string, permission *json.RawMessage) error
}
