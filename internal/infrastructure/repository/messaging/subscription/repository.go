package subscription

import (
	"errors"
	"github.com/qpubio/qpub-server/internal/domain/messaging/subscription"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/clock"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"sync"
	"time"
)

type repository struct {
	logger        logger.Service
	instanceID    id.ULID
	subscriptions map[string]map[*subscription.Subscription]struct{} // channelName -> subscription map
	subChannels   map[id.ULID]map[string]struct{}                    // subscriptionID -> channel names (reverse index)
	mu            sync.RWMutex
}

// NewRepository creates a new subscriber repository instance
func NewRepository(logger logger.Service, instanceID id.ULID) subscription.Repository {
	return &repository{
		logger:        logger,
		instanceID:    instanceID,
		subscriptions: make(map[string]map[*subscription.Subscription]struct{}),
		subChannels:   make(map[id.ULID]map[string]struct{}),
	}
}

// AddToChannel adds a subscription to a channel
func (r *repository) AddToChannel(channelName string, sub *subscription.Subscription) error {
	if sub == nil {
		return subscription.ErrNil
	}

	if sub.IsClosed() {
		return subscription.ErrClosed
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize channel's subscriber map if it doesn't exist
	if _, ok := r.subscriptions[channelName]; !ok {
		r.subscriptions[channelName] = make(map[*subscription.Subscription]struct{})
	}

	// Add subscriber to channel
	r.subscriptions[channelName][sub] = struct{}{}

	// Maintain reverse index: subscription -> channels
	if _, ok := r.subChannels[sub.ID()]; !ok {
		r.subChannels[sub.ID()] = make(map[string]struct{})
	}
	r.subChannels[sub.ID()][channelName] = struct{}{}

	// Update subscription activity time
	sub.UpdateActivity()

	return nil
}

// RemoveFromChannel removes a subscription from a channel
func (r *repository) RemoveFromChannel(channelName string, sub *subscription.Subscription) error {
	if sub == nil {
		return subscription.ErrNil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if channel exists
	channelSubs, ok := r.subscriptions[channelName]
	if !ok {
		// Channel not found, but still clean reverse index to prevent leaks
		if channels, exists := r.subChannels[sub.ID()]; exists {
			delete(channels, channelName)
			if len(channels) == 0 {
				delete(r.subChannels, sub.ID())
			}
		}
		return errors.New("channel not found")
	}

	// Remove subscription from channel
	delete(channelSubs, sub)

	// Clean up reverse index
	if channels, exists := r.subChannels[sub.ID()]; exists {
		delete(channels, channelName)
		if len(channels) == 0 {
			delete(r.subChannels, sub.ID())
		}
	}

	remainingSubsInChannel := len(channelSubs)

	// If channel has no more subscriptions, remove the channel
	if remainingSubsInChannel == 0 {
		delete(r.subscriptions, channelName)
	}

	return nil
}

// RemoveSubscriptionFromAll removes a subscription from all its channels atomically
// This is critical for preventing race conditions during disconnection
func (r *repository) RemoveSubscriptionFromAll(sub *subscription.Subscription) error {
	if sub == nil {
		return subscription.ErrNil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Get all channels for this subscription from reverse index
	channelNames, exists := r.subChannels[sub.ID()]
	if !exists {
		// No channels found, subscription not in repository
		return nil
	}

	// Remove from all channels
	for channelName := range channelNames {
		if channelSubs, ok := r.subscriptions[channelName]; ok {
			delete(channelSubs, sub)

			// If channel has no more subscriptions, remove the channel
			if len(channelSubs) == 0 {
				delete(r.subscriptions, channelName)
			}
		}
	}

	// Clean up reverse index
	delete(r.subChannels, sub.ID())

	return nil
}

// GetAllLocalFromChannel retrieves all subscriptions for a channel on this instance
func (r *repository) GetAllLocalForChannel(channelName string) ([]*subscription.Subscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if channel exists
	channelSubs, ok := r.subscriptions[channelName]
	if !ok {
		return nil, subscription.ErrChannelNotFound
	}

	// Convert map to slice, filtering out closed subscriptions
	subscriptions := make([]*subscription.Subscription, 0, len(channelSubs))
	for sub := range channelSubs {
		if !sub.IsClosed() {
			// Update activity time when accessed
			sub.UpdateActivity()
			subscriptions = append(subscriptions, sub)
		}
	}

	return subscriptions, nil
}

// GetAllChannelsForSubscription retrieves all channels a subscription is subscribed to
// Now uses reverse index for O(1) lookup instead of O(n) iteration
func (r *repository) GetAllChannelsForSubscription(sub *subscription.Subscription) ([]string, error) {
	if sub == nil {
		return nil, subscription.ErrNil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Use reverse index for efficient lookup
	channelMap, exists := r.subChannels[sub.ID()]
	if !exists {
		return []string{}, nil
	}

	// Convert map to slice
	channels := make([]string, 0, len(channelMap))
	for channelName := range channelMap {
		channels = append(channels, channelName)
	}

	// Update activity time when accessed
	if len(channels) > 0 {
		sub.UpdateActivity()
	}

	return channels, nil
}

// CleanClosedSubscriptions removes all closed subscriptions from all channels
// Returns the number of closed subscriptions removed
func (r *repository) CleanClosedSubscriptions() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	removedCount := 0
	now := clock.Now()
	idleTimeout := 5 * time.Minute

	// Iterate through all channels
	for channelName, subs := range r.subscriptions {
		// Remove closed or idle subscriptions
		for sub := range subs {
			if sub.IsClosed() || now.Sub(sub.LastActive()) > idleTimeout {
				delete(subs, sub)
				removedCount++

				// Clean up reverse index
				if channels, exists := r.subChannels[sub.ID()]; exists {
					delete(channels, channelName)
					if len(channels) == 0 {
						delete(r.subChannels, sub.ID())
					}
				}

				if !sub.IsClosed() {
					// If removing due to idle timeout, close the subscription
					sub.Close()
					r.logger.Info(log.MessagingSubscription, `Closed idle subscription subscriptionID=%v channel=%v idleTime=%v`, sub.ID(),
						channelName,
						now.Sub(sub.LastActive()).String())
				}
			}
		}

		// If channel has no more subscriptions, remove it
		if len(subs) == 0 {
			delete(r.subscriptions, channelName)
			r.logger.Debug(log.MessagingSubscription, `Channel removed during cleanup (no more subscriptions) channel=%v instanceID=%v`, channelName,
				r.instanceID)
		}
	}

	if removedCount > 0 {
		r.logger.Info(log.MessagingSubscription, `Cleaned closed/idle subscriptions count=%v instanceID=%v`, removedCount,
			r.instanceID)
	}

	return removedCount
}

// FindByID retrieves a subscription by its ID
func (r *repository) FindByID(subID id.ULID) (*subscription.Subscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, subs := range r.subscriptions {
		for sub := range subs {
			if sub.ID() == subID {
				return sub, nil
			}
		}
	}

	return nil, errors.New("subscription not found")
}
