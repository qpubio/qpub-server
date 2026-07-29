package broker

import (
	"context"
	"fmt"
	"github.com/qpubio/qpub-server/internal/domain/messaging/broker"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type repository struct {
	natsConn      *nats.Conn
	subscriptions map[string]*nats.Subscription
	handlers      map[string]broker.MessageHandler
	mu            sync.RWMutex
	logger        logger.Service
}

// NewRepository creates a new broker repository instance using NATS
func NewRepository(nc *nats.Conn, logger logger.Service) broker.Repository {
	return &repository{
		natsConn:      nc,
		subscriptions: make(map[string]*nats.Subscription),
		handlers:      make(map[string]broker.MessageHandler),
		logger:        logger,
	}
}

// Subscribe registers a handler for messages on a channel
func (r *repository) Subscribe(channelName string, handler broker.MessageHandler) error {
	if r.natsConn == nil || !r.natsConn.IsConnected() {
		return broker.ErrBrokerUnavailable
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already subscribed
	if _, exists := r.subscriptions[channelName]; exists {
		r.logger.Warn(log.MessagingBroker, `Already subscribed to channel channel=%v`, channelName)

		// Update the handler if it's different
		r.handlers[channelName] = handler
		return nil
	}

	// Create NATS subscription
	subscription, err := r.natsConn.Subscribe(channelName, func(msg *nats.Msg) {
		// Get the handler with read lock
		r.mu.RLock()
		msgHandler, exists := r.handlers[channelName]
		r.mu.RUnlock()

		if exists && msgHandler != nil {
			// Call the handler with the message
			msgHandler(channelName, msg.Data)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to NATS channel %s: %w", channelName, err)
	}

	// Store subscription and handler
	r.subscriptions[channelName] = subscription
	r.handlers[channelName] = handler

	return nil
}

// Unsubscribe removes a handler for messages on a channel
func (r *repository) Unsubscribe(channelName string) error {
	if r.natsConn == nil {
		return broker.ErrBrokerUnavailable
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if subscribed
	subscription, exists := r.subscriptions[channelName]
	if !exists {
		return nil
	}

	// Unsubscribe from NATS
	if err := subscription.Unsubscribe(); err != nil {
		return fmt.Errorf("failed to unsubscribe from NATS channel %s: %w", channelName, err)
	}

	// Remove subscription and handler
	delete(r.subscriptions, channelName)
	delete(r.handlers, channelName)

	return nil
}

// Publish sends a message to a channel
func (r *repository) Publish(channelName string, message []byte) error {
	if r.natsConn == nil || !r.natsConn.IsConnected() {
		return broker.ErrBrokerUnavailable
	}

	// Publish to NATS
	if err := r.natsConn.Publish(channelName, message); err != nil {
		return fmt.Errorf("failed to publish message to NATS channel %s: %w", channelName, err)
	}

	return nil
}

// Close gracefully closes all broker connections
func (r *repository) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Determine cleanup timeout
	timeout := 5 * time.Second // Default timeout

	// If context has a deadline and it's in the future, use remaining time
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		remaining := time.Until(deadline)
		if remaining > 0 {
			timeout = remaining
		}
		// If deadline is in the past (context already cancelled), use default timeout
	}

	// Create a fresh context with timeout for clean shutdown
	// This ensures cleanup happens even if parent context is cancelled
	cleanupCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Unsubscribe from all subscriptions
	for channelName, subscription := range r.subscriptions {
		select {
		case <-cleanupCtx.Done():
			return fmt.Errorf("broker shutdown timed out")
		default:
			if err := subscription.Unsubscribe(); err != nil {
				r.logger.Error(log.MessagingBroker, `Error unsubscribing during shutdown channel=%v error=%v`, channelName,
					err)
				// Continue with other unsubscriptions even if one fails
			}
		}
	}

	// Clear maps
	r.subscriptions = make(map[string]*nats.Subscription)
	r.handlers = make(map[string]broker.MessageHandler)

	return nil
}
