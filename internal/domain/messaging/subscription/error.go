package subscription

import "errors"

var (
	ErrNil             = errors.New("subscription cannot be nil")
	ErrClosed          = errors.New("subscription is closed")
	ErrChannelNotFound = errors.New("channel not found")
)
