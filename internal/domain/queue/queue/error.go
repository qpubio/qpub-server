package queue

import "errors"

var (
	ErrNotFound      = errors.New("queue not found")
	ErrAlreadyExists = errors.New("queue already exists")
)
