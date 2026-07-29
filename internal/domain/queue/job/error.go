package job

import "errors"

var (
	ErrNotFound          = errors.New("job not found")
	ErrNotClaimable      = errors.New("job is not claimable")
	ErrInvalidTransition = errors.New("invalid job state transition")
	ErrWorkerMismatch    = errors.New("worker does not own this job")
)
