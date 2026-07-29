package connection

import "errors"

var (
	// ErrConnectionNotFound is returned when a connection is not found
	ErrConnectionNotFound = errors.New("connection not found")

	// ErrConnectionClosed is returned when a connection is closed
	ErrConnectionClosed = errors.New("connection closed")

	// ErrSendFailed is returned when a message send fails
	ErrSendFailed = errors.New("failed to send message")

	// ErrConnectionLimit is returned when a connection limit is reached
	ErrConnectionLimit = errors.New("connection limit reached")
)
