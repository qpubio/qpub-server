package subscription

import "github.com/qpubio/qpub-server/internal/shared/id"

// Repository defines the data access contract for subscriptions
type Repository interface {
	// AddToChannel adds a subscription to a channel
	AddToChannel(channelName string, sub *Subscription) error

	// RemoveFromChannel removes a subscription from a channel
	RemoveFromChannel(channelName string, sub *Subscription) error

	// RemoveSubscriptionFromAll removes a subscription from all channels atomically
	RemoveSubscriptionFromAll(sub *Subscription) error

	// GetAllLocalForChannel retrieves all active subscriptions for a channel
	GetAllLocalForChannel(channelName string) ([]*Subscription, error)

	// GetAllChannelsForSubscription retrieves all channels a subscription is subscribed to
	GetAllChannelsForSubscription(sub *Subscription) ([]string, error)

	// CleanClosedSubscriptions removes all closed subscriptions from all channels
	CleanClosedSubscriptions() int

	// FindByID retrieves a subscription by its ID
	FindByID(subID id.ULID) (*Subscription, error)
}
