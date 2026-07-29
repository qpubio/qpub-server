package broker

import "errors"

var (
	// ErrSubscriptionFailed is returned when subscription to a channel fails
	ErrSubscriptionFailed = errors.New("failed to subscribe to channel")

	// ErrUnsubscriptionFailed is returned when unsubscription from a channel fails
	ErrUnsubscriptionFailed = errors.New("failed to unsubscribe from channel")

	// ErrPublishFailed is returned when publishing to a channel fails
	ErrPublishFailed = errors.New("failed to publish to channel")

	// ErrChannelNotFound is returned when the requested channel doesn't exist
	ErrChannelNotFound = errors.New("channel not found")

	// ErrBrokerUnavailable is returned when the message broker is unavailable
	ErrBrokerUnavailable = errors.New("message broker is unavailable")
)
