package queue

import "errors"

var (
	ErrNotFound         = errors.New("queue not found")
	ErrAlreadyExists    = errors.New("queue already exists")
	ErrDeleting         = errors.New("queue is deleting")
	ErrActiveJobs       = errors.New("queue has active jobs")
)
