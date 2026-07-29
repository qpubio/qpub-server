package channel

import "errors"

var (
	ErrNotFound                 = errors.New("channel not found")
	ErrBrokerSubscriptionFailed = errors.New("failed to subscribe to broker")
	ErrAlreadyExists            = errors.New("channel already exists")
)
