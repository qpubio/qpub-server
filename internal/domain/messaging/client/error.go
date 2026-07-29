package client

import "errors"

var (
	// ErrClientNotFound is returned when a client is not found
	ErrClientNotFound = errors.New("client not found")

	// ErrClientAlreadyExists is returned when a client already exists
	ErrClientAlreadyExists = errors.New("client already exists")

	// ErrClientDisconnected is returned when a client is disconnected
	ErrClientDisconnected = errors.New("client disconnected")

	// ErrInvalidClientID is returned when a client ID is invalid
	ErrInvalidClientID = errors.New("invalid client ID")

	// ErrClientIDAlreadyExists is returned when a client ID already exists
	ErrClientIDAlreadyExists = errors.New("client ID already exists")

	// ErrClientUnauthenticated is returned when a client is not authenticated
	ErrClientUnauthenticated = errors.New("client unauthenticated")
)
