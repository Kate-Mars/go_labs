package trip

import "errors"

var (
	ErrNotFound            = errors.New("trip not found")
	ErrDriverBusy          = errors.New("driver already has an active trip")
	ErrAlreadyDone         = errors.New("trip already completed")
	ErrIdempotencyConflict = errors.New("idempotency key was used with a different request body")
)
