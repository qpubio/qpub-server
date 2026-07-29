package subscription

import (
	"github.com/qpubio/qpub-server/internal/shared/id"
)

// Service defines the subscription domain service contract
type Service interface {
	// Subscribe adds a subscription to a channel
	Subscribe(channelRawName string, sub *Subscription, projID id.Int) error

	// Unsubscribe removes a subscription from a channel
	Unsubscribe(channelRawName string, sub *Subscription, projID id.Int) error

	// Get returns a subscription by its ID
	Get(subID id.ULID) (*Subscription, error)

	// GetAllLocalForChannel returns all local subscriptions for a channel
	GetAllLocalForChannel(channelRawName string) ([]*Subscription, error)

	// GetSubscriptions returns all channels the subscription is subscribed to
	GetSubscriptions(sub *Subscription) ([]string, error)

	// CloseSubscription marks a subscription as closed and cleans up resources
	CloseSubscription(sub *Subscription) error
}
