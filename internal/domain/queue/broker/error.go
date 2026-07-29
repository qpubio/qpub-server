package broker

import "errors"

var (
	ErrBrokerUnavailable = errors.New("queue broker unavailable")
	ErrStreamNotFound      = errors.New("queue stream not found")
)
