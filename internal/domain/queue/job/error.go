package job

import "errors"

var (
	ErrNotFound             = errors.New("job not found")
	ErrNotClaimable         = errors.New("job is not claimable")
	ErrInvalidTransition    = errors.New("invalid job state transition")
	ErrWorkerMismatch       = errors.New("worker does not own this job")
	ErrInvalidTerminalState = errors.New("invalid terminal_at for job status")
	ErrTerminalJob          = errors.New("cannot update progress on terminal job")
)
