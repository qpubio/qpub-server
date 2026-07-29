package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/qpubio/qpub-server/internal/domain/messaging/channel"
	"github.com/qpubio/qpub-server/internal/domain/messaging/delivery"
	"github.com/qpubio/qpub-server/internal/domain/messaging/event"
	"github.com/qpubio/qpub-server/internal/domain/messaging/protocol"
	domainRouter "github.com/qpubio/qpub-server/internal/domain/messaging/router"
	"github.com/qpubio/qpub-server/internal/domain/messaging/subscription"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

// Service is the enhanced subscription service with event-driven architecture
type Service struct {
	logger         logger.Service
	instanceID     id.ULID
	repository     subscription.Repository
	channelService channel.Service
	router         domainRouter.Service
	deliverer      delivery.Deliverer
	eventBus       event.Service
}

func NewService(
	logger logger.Service,
	instanceID id.ULID,
	repository subscription.Repository,
	channelService channel.Service,
	router domainRouter.Service,
	deliverer delivery.Deliverer,
	eventBus event.Service,
) subscription.Service {
	return &Service{
		logger:         logger,
		instanceID:     instanceID,
		repository:     repository,
		channelService: channelService,
		router:         router,
		deliverer:      deliverer,
		eventBus:       eventBus,
	}
}

func (s *Service) Subscribe(channelRawName string, sub *subscription.Subscription, projID id.Int) error {
	if sub.IsClosed() {
		errMsg := "subscription is closed"
		s.sendSubscriptionState(
			protocol.ActionSubscribe,
			sub,
			channelRawName,
			protocol.ErrSubscriptionClosed,
			protocol.HrefSubscriptionClosed,
			errMsg,
			protocol.StatusCodeBadRequest,
		)
		return subscription.ErrClosed
	}

	// Get or create the channel
	ch, err := s.channelService.GetOrCreate(channelRawName, projID)
	if err != nil || ch == nil {
		s.sendSubscriptionState(
			protocol.ActionSubscribe,
			sub,
			channelRawName,
			protocol.ErrInternal,
			protocol.HrefInternal,
			"Failed to get or create channel",
			protocol.StatusCodeInternal,
		)
		s.logger.Error(log.MessagingSubscription, `Failed to get or create channel subscriptionID=%v channel=%v projectID=%v error=%v`, sub.ID(),
			channelRawName,
			projID,
			err)
		return err
	}

	// Check if already subscribed
	subscribedChannels, err := s.repository.GetAllChannelsForSubscription(sub)
	if err != nil {
		s.logger.Error(log.MessagingSubscription, `Failed to get current channels for subscription check subscriptionID=%v error=%v`, sub.ID(), err)
	} else {
		for _, existingChannel := range subscribedChannels {
			if existingChannel == ch.FullName() {
				s.logger.Info(log.MessagingSubscription, `Subscription already exists for this channel subscriptionID=%v channel=%v fullName=%v`, sub.ID(),
					channelRawName,
					ch.FullName())
				s.sendSubscriptionState(
					protocol.ActionSubscribed,
					sub,
					channelRawName,
					0, "", "", 0,
				)
				return nil
			}
		}
	}

	// Add subscription to channel - IN-MEMORY ONLY
	err = s.repository.AddToChannel(ch.FullName(), sub)
	if err != nil {
		s.sendSubscriptionState(
			protocol.ActionSubscribe,
			sub,
			channelRawName,
			protocol.ErrSubscribeFailed,
			protocol.HrefSubscribeFailed,
			"Failed to subscribe to channel",
			protocol.StatusCodeInternal,
		)
		s.logger.Error(log.MessagingSubscription, `Failed to add subscription to channel subscriptionID=%v channel=%v projectID=%v error=%v`, sub.ID(),
			channelRawName,
			projID,
			err)
		return err
	}

	if err := s.router.EnsureChannelListening(ch.FullName()); err != nil {
		s.sendSubscriptionState(
			protocol.ActionSubscribe,
			sub,
			channelRawName,
			protocol.ErrInternal,
			protocol.HrefInternal,
			"Failed to set up channel delivery",
			protocol.StatusCodeInternal,
		)
		s.logger.Error(log.MessagingSubscription, "Failed to ensure channel listening subscriptionID=%s channel=%s error=%v",
			sub.ID(), channelRawName, err)
		return err
	}

	// Update channel's internal counter for tracking
	ch.IncrementLocalSubscriptions()
	if err := s.channelService.Update(ch); err != nil {
		s.logger.Error(log.MessagingSubscription, `Failed to update channel subscription count subscriptionID=%v channel=%v fullName=%v error=%v`, sub.ID(),
			channelRawName,
			ch.FullName(),
			err)
	}

	s.logger.Info(log.MessagingSubscription, `Subscription added to channel subscriptionID=%v channel=%v fullName=%v projectID=%v`, sub.ID(),
		channelRawName,
		ch.FullName(),
		projID)

	// Send success response
	s.sendSubscriptionState(
		protocol.ActionSubscribed,
		sub,
		channelRawName,
		0, "", "", 0,
	)

	// Publish channel subscribed event for logging and lifecycle management
	evt := event.NewEvent(event.EventChannelSubscribed, event.ChannelSubscribedData{
		ChannelName:    channelRawName,
		ProjectID:      projID,
		ClientID:       sub.ClientID(),
		SubscriptionID: sub.ID(),
	})

	if err := s.eventBus.Publish(context.Background(), evt); err != nil {
		s.logger.Error(log.MessagingSubscription, `Failed to publish channel subscribed event error=%v`, err)
	}

	return nil
}

func (s *Service) Unsubscribe(channelRawName string, sub *subscription.Subscription, projID id.Int) error {
	// Build full channel name for repository operations
	ch, err := s.channelService.Get(channelRawName, projID)
	fullChannelName := ""

	if err == nil && ch != nil {
		fullChannelName = ch.FullName()
	} else if err == channel.ErrNotFound {
		// Channel was deleted, but we can still construct the full name
		channelName := channel.NewName(channelRawName, projID)
		fullChannelName = channelName.Full()
	} else if err != nil {
		// Unexpected error, but don't fail - try to remove anyway
		s.logger.Warn(log.MessagingSubscription, `Error getting channel during unsubscribe, will attempt removal anyway subscriptionID=%v channel=%v projectID=%v error=%v`, sub.ID(),
			channelRawName,
			projID,
			err)
	}

	// CRITICAL: Always remove subscription from repository, even if channel is nil/deleted
	// This prevents subscription leaks when channels are deleted before unsubscribe
	if fullChannelName != "" {
		err = s.repository.RemoveFromChannel(fullChannelName, sub)
		if err != nil && err.Error() != "channel not found" {
			s.logger.Warn(log.MessagingSubscription, `Error removing subscription from repository subscriptionID=%v channel=%v fullName=%v error=%v`, sub.ID(),
				channelRawName,
				fullChannelName,
				err)
			// Don't return error - continue with cleanup
		}
	}

	// Update channel's internal counter if channel exists
	if ch != nil {
		ch.DecrementLocalSubscriptions()
		if err := s.channelService.Update(ch); err != nil {
			s.logger.Error(log.MessagingSubscription, `Failed to update channel subscription count subscriptionID=%v channel=%v fullName=%v error=%v`, sub.ID(),
				channelRawName,
				fullChannelName,
				err)
		}
	}

	s.logger.Info(log.MessagingSubscription, `Subscription removed from channel subscriptionID=%v channel=%v projectID=%v`, sub.ID(),
		channelRawName,
		projID)

	s.sendUnsubscriptionState(
		protocol.ActionUnsubscribed,
		sub,
		channelRawName,
		0, "", "", 0,
	)

	// Publish channel unsubscribed event for logging and lifecycle management
	evt := event.NewEvent(event.EventChannelUnsubscribed, event.ChannelUnsubscribedData{
		ChannelName:    channelRawName,
		ProjectID:      projID,
		ClientID:       sub.ClientID(),
		SubscriptionID: sub.ID(),
	})

	if err := s.eventBus.Publish(context.Background(), evt); err != nil {
		s.logger.Error(log.MessagingSubscription, `Failed to publish channel unsubscribed event error=%v`, err)
	}

	return nil
}

func (s *Service) CloseSubscription(sub *subscription.Subscription) error {
	if sub.IsClosed() {
		return nil // Already closed
	}

	// Get all channels this subscription is subscribed to BEFORE marking as closed
	channels, err := s.repository.GetAllChannelsForSubscription(sub)
	if err != nil {
		s.logger.Error(log.MessagingSubscription, `Error getting channels during subscription closure subscriptionID=%v error=%v`, sub.ID(),
			err)
		// Continue anyway to ensure subscription is closed
	}

	// CRITICAL: Mark subscription as closed FIRST (atomic operation)
	// This ensures snapshot service immediately stops counting this subscription
	// even if removal from repository takes time
	sub.Close()

	// CRITICAL: Remove from repository atomically, using direct removal
	// This doesn't depend on channel lookups and can't fail due to missing channels
	if err := s.repository.RemoveSubscriptionFromAll(sub); err != nil {
		s.logger.Error(log.MessagingSubscription, `Error removing subscription from repository subscriptionID=%v error=%v`, sub.ID(),
			err)
		// Continue with cleanup even if removal fails
	}

	// Cleanup: Unregister broker listeners and update channel stats
	// These are "nice to have" cleanups that don't affect subscriber counts
	for _, fullChannelName := range channels {
		// Parse the full channel name to get components
		channelName, err := channel.FromFull(fullChannelName)
		if err != nil {
			s.logger.Error(log.MessagingSubscription, `Error parsing full channel name during closure fullName=%v error=%v`, fullChannelName,
				err)
			continue
		}

		// Update channel's internal counter (optional, for channel lifecycle)
		ch, err := s.channelService.Get(channelName.Raw(), channelName.ProjectID())
		if err == nil && ch != nil {
			ch.DecrementLocalSubscriptions()
			if err := s.channelService.Update(ch); err != nil {
				s.logger.Error(log.MessagingSubscription, `Failed to update channel subscription count subscriptionID=%v channel=%v fullName=%v error=%v`, sub.ID(),
					channelName.Raw(),
					fullChannelName,
					err)
			}
		}

		// Publish channel unsubscribed event for logging
		evt := event.NewEvent(event.EventChannelUnsubscribed, event.ChannelUnsubscribedData{
			ChannelName:    channelName.Raw(),
			ProjectID:      channelName.ProjectID(),
			ClientID:       sub.ClientID(),
			SubscriptionID: sub.ID(),
		})
		if err := s.eventBus.Publish(context.Background(), evt); err != nil {
			s.logger.Error(log.MessagingSubscription, `Failed to publish channel unsubscribed event error=%v`, err)
		}
	}

	// Publish subscription closed event
	evt := event.NewEvent(event.EventSubscriptionClosed, event.SubscriptionClosedData{
		SubscriptionID: sub.ID(),
		ClientID:       sub.ClientID(),
		ChannelCount:   len(channels),
	})

	if err := s.eventBus.Publish(context.Background(), evt); err != nil {
		s.logger.Error(log.MessagingSubscription, `Failed to publish subscription closed event error=%v`, err)
	}

	s.logger.Info(log.MessagingSubscription, `Subscription closed successfully subscriptionID=%v channelCount=%v`, sub.ID(),
		len(channels))

	return nil
}

// Delegate methods to maintain interface compatibility
func (s *Service) Get(subID id.ULID) (*subscription.Subscription, error) {
	return s.repository.FindByID(subID)
}

func (s *Service) GetAllLocalForChannel(channelName string) ([]*subscription.Subscription, error) {
	subscriptions, err := s.repository.GetAllLocalForChannel(channelName)
	if err != nil {
		s.logger.Error(log.MessagingSubscription, "Error getting all local subscriptions for channel: %v", err)
		return nil, err
	}
	return subscriptions, nil
}

func (s *Service) GetSubscriptions(sub *subscription.Subscription) ([]string, error) {
	fullChannelNames, err := s.repository.GetAllChannelsForSubscription(sub)
	if err != nil {
		s.logger.Error(log.MessagingSubscription, `Error getting channels for subscription subscriptionID=%v error=%v`, sub.ID(),
			err)
		return nil, err
	}

	// Convert full channel names to client-facing channel names
	channelNames := make([]string, 0, len(fullChannelNames))
	for _, fullName := range fullChannelNames {
		channelName, err := channel.FromFull(fullName)
		if err != nil {
			s.logger.Error(log.MessagingSubscription, `Error parsing full channel name fullName=%v error=%v`, fullName,
				err)
			continue
		}
		channelNames = append(channelNames, channelName.Raw())
	}

	return channelNames, nil
}

func (s *Service) sendSubscriptionState(
	action protocol.ActionType,
	sub *subscription.Subscription,
	channelName string,
	errCode protocol.Code,
	errHref protocol.Href,
	errMessage string,
	errStatusCode protocol.StatusCode,
) error {
	if sub.IsClosed() {
		return nil
	}

	var errorInfo *protocol.ErrorInfo
	if errCode != 0 {
		errorInfo = protocol.NewErrorInfo(
			int(errCode),
			string(errHref),
			errMessage,
			int(errStatusCode),
		)
	}

	stateMsg := protocol.NewChannelMessage(
		action,
		channelName,
		sub.ID(),
		errorInfo,
	)

	msgBytes, err := json.Marshal(stateMsg)
	if err != nil {
		s.logger.Error(log.MessagingSubscription, `Error marshaling subscription state message subscriptionID=%v channel=%v error=%v`, sub.ID(),
			channelName,
			err)
		return fmt.Errorf("error marshaling subscription state message: %w", err)
	}

	if err := s.deliverer.Deliver(sub.ClientID(), msgBytes); err != nil {
		s.logger.Warn(log.MessagingSubscription, `Failed to deliver subscription state message subscriptionID=%v channel=%v clientID=%v error=%v`, sub.ID(),
			channelName,
			sub.ClientID(),
			err)
	} else {
		s.logger.Debug(log.MessagingSubscription, `Sent subscription state message subscriptionID=%v channel=%v action=%v`, sub.ID(),
			channelName,
			protocol.ActionToString(action))
	}

	return nil
}

func (s *Service) sendUnsubscriptionState(
	action protocol.ActionType,
	sub *subscription.Subscription,
	channelName string,
	errCode protocol.Code,
	errHref protocol.Href,
	errMessage string,
	errStatusCode protocol.StatusCode,
) error {
	return s.sendSubscriptionState(action, sub, channelName, errCode, errHref, errMessage, errStatusCode)
}
