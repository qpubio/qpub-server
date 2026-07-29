package channel

import (
	"context"
	"github.com/qpubio/qpub-server/internal/domain/messaging/broker"
	"github.com/qpubio/qpub-server/internal/domain/messaging/channel"
	"github.com/qpubio/qpub-server/internal/domain/messaging/event"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

// Service is the enhanced channel service with event-driven architecture
type Service struct {
	logger        logger.Service
	instanceID    id.ULID
	repository    channel.Repository
	brokerService broker.Service
	eventBus      event.Service
}

func NewService(
	logger logger.Service,
	instanceID id.ULID,
	repository channel.Repository,
	brokerService broker.Service,
	eventBus event.Service,
) channel.Service {
	return &Service{
		logger:        logger,
		instanceID:    instanceID,
		repository:    repository,
		brokerService: brokerService,
		eventBus:      eventBus,
	}
}

func (s *Service) Create(rawName string, projID id.Int) (*channel.Channel, error) {
	channelName := channel.NewName(rawName, projID)

	// Check if the channel already exists
	existingChannel, err := s.repository.FindByName(channelName.Full())
	if err == nil {
		return existingChannel, nil // Channel found, return it
	}
	if err != channel.ErrNotFound {
		return nil, err // Return any error that isn't ErrNotFound
	}

	// Create new channel
	ch := channel.New(channelName, s.instanceID)

	// Store in repository
	if err := s.repository.Create(ch); err != nil {
		return nil, err
	}

	// Mark channel as active
	ch.SetActive(true)
	s.repository.Update(ch)

	s.logger.Info(log.MessagingChannel, `Channel created channel=%v fullName=%v projectID=%v instanceID=%v`, rawName,
		channelName.Full(),
		projID,
		s.instanceID)

	// Publish channel created event
	evt := event.NewEvent(event.EventChannelCreated, event.ChannelCreatedData{
		ChannelID:   ch.ID(),
		ChannelName: rawName,
		ProjectID:   projID,
		InstanceID:  s.instanceID,
	})

	if err := s.eventBus.Publish(context.Background(), evt); err != nil {
		s.logger.Error(log.MessagingChannel, `Failed to publish channel created event error=%v`, err)
	}

	return ch, nil
}

func (s *Service) Update(ch *channel.Channel) error {
	return s.repository.Update(ch)
}

func (s *Service) Delete(rawName string, projID id.Int) error {
	channelName := channel.NewName(rawName, projID)
	fullName := channelName.Full()

	// Get the channel before deletion
	ch, err := s.repository.FindByName(fullName)
	if err != nil {
		if err == channel.ErrNotFound {
			return nil // Already deleted
		}
		return err
	}

	// Only delete if no local subscribers
	if ch.HasLocalSubscribers() {
		s.logger.Debug(log.MessagingChannel, `Cannot delete channel with active subscribers channel=%v fullName=%v subscriberCount=%v`, rawName,
			fullName,
			ch.LocalSubscriptionCount())
		return nil // Don't delete if there are still subscribers
	}

	// Stop broker from listening to this channel
	if err := s.brokerService.StopListeningToChannel(fullName); err != nil {
		s.logger.Error(log.MessagingChannel, `Failed to stop broker listener for channel channel=%v fullName=%v error=%v`, rawName,
			fullName,
			err)
		// Continue anyway
	}

	// Delete from repository
	if err := s.repository.Delete(ch); err != nil {
		return err
	}

	s.logger.Info(log.MessagingChannel, `Channel deleted channel=%v fullName=%v projectID=%v instanceID=%v`, rawName,
		fullName,
		projID,
		s.instanceID)

	return nil
}

func (s *Service) Get(rawName string, projID id.Int) (*channel.Channel, error) {
	channelName := channel.NewName(rawName, projID)
	return s.repository.FindByName(channelName.Full())
}

func (s *Service) GetOrCreate(rawName string, projID id.Int) (*channel.Channel, error) {
	// Try to get existing channel
	ch, err := s.Get(rawName, projID)
	if err == nil {
		return ch, nil
	}

	// If not found, create new channel
	if err == channel.ErrNotFound {
		return s.Create(rawName, projID)
	}

	// Return other errors
	return nil, err
}
