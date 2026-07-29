package broker

import (
	"context"
	"github.com/qpubio/qpub-server/internal/domain/messaging/broker"
	"github.com/qpubio/qpub-server/internal/infrastructure/logger"
	"github.com/qpubio/qpub-server/internal/shared/id"
	"github.com/qpubio/qpub-server/internal/shared/type/log"
)

type Service struct {
	repository broker.Repository
	instanceID id.ULID
	logger     logger.Service
}

// NewService creates a new broker service
func NewService(repository broker.Repository, instanceID id.ULID, logger logger.Service) broker.Service {
	return &Service{
		repository: repository,
		instanceID: instanceID,
		logger:     logger,
	}
}

// ListenToChannel registers a handler for messages on a channel from the broker
func (s *Service) ListenToChannel(channelName string, handler broker.MessageHandler) error {
	s.logger.Info(log.MessagingBroker, `Setting up broker listener channel=%v instanceID=%v`, channelName,
		s.instanceID)

	err := s.repository.Subscribe(channelName, handler)
	if err != nil {
		s.logger.Error(log.MessagingBroker, `Failed to subscribe to broker channel channel=%v instanceID=%v error=%v`, channelName,
			s.instanceID,
			err)
		return broker.ErrSubscriptionFailed
	}

	return nil
}

// StopListeningToChannel removes a handler for messages on a channel
func (s *Service) StopListeningToChannel(channelName string) error {
	s.logger.Info(log.MessagingBroker, `Removing broker listener channel=%v instanceID=%v`, channelName,
		s.instanceID)

	err := s.repository.Unsubscribe(channelName)
	if err != nil {
		s.logger.Error(log.MessagingBroker, `Failed to unsubscribe from broker channel channel=%v instanceID=%v error=%v`, channelName,
			s.instanceID,
			err)
		return broker.ErrUnsubscriptionFailed
	}

	return nil
}

// PublishToChannel publishes a message to a channel via the broker
func (s *Service) PublishToChannel(channelName string, message []byte) error {
	s.logger.Debug(log.MessagingBroker, `Publishing message to broker channel=%v messageSize=%v instanceID=%v`, channelName,
		len(message),
		s.instanceID)

	err := s.repository.Publish(channelName, message)
	if err != nil {
		s.logger.Error(log.MessagingBroker, `Failed to publish message channel=%v instanceID=%v error=%v`, channelName,
			s.instanceID,
			err)
		return broker.ErrPublishFailed
	}

	return nil
}

// Shutdown gracefully closes all broker connections
func (s *Service) Shutdown(ctx context.Context) error {
	s.logger.Info(log.MessagingBroker, `Shutting down broker service instanceID=%v`, s.instanceID)

	err := s.repository.Close(ctx)
	if err != nil {
		s.logger.Error(log.MessagingBroker, `Error during broker shutdown instanceID=%v error=%v`, s.instanceID,
			err)
		return err
	}

	return nil
}
