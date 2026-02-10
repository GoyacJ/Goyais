package command

import "errors"

var (
	ErrNotFound              = errors.New("command not found")
	ErrNotImplemented        = errors.New("command repository not implemented")
	ErrIdempotencyConflict   = errors.New("idempotency key conflict")
	ErrInvalidCursor         = errors.New("invalid cursor")
	ErrInvalidCommandRequest = errors.New("invalid command request")
)

type IdempotencyConflictError struct {
	ExistingCommandID string
}

func (e *IdempotencyConflictError) Error() string {
	if e == nil || e.ExistingCommandID == "" {
		return ErrIdempotencyConflict.Error()
	}
	return ErrIdempotencyConflict.Error() + ": " + e.ExistingCommandID
}
