package command

import "errors"

var (
	ErrNotFound              = errors.New("command not found")
	ErrShareNotFound         = errors.New("share not found")
	ErrNotImplemented        = errors.New("command repository not implemented")
	ErrIdempotencyConflict   = errors.New("idempotency key conflict")
	ErrInvalidCursor         = errors.New("invalid cursor")
	ErrInvalidCommandRequest = errors.New("invalid command request")
	ErrInvalidShareRequest   = errors.New("invalid share request")
	ErrForbidden             = errors.New("forbidden")
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

type ForbiddenError struct {
	Reason string
}

func (e *ForbiddenError) Error() string {
	if e == nil || e.Reason == "" {
		return ErrForbidden.Error()
	}
	return ErrForbidden.Error() + ": " + e.Reason
}

func (e *ForbiddenError) Is(target error) bool {
	return target == ErrForbidden
}

// ExecutionError carries domain-specific failure metadata from command executors.
// It unwraps to a canonical command error so existing error matching still works.
type ExecutionError struct {
	Code       string
	MessageKey string
	Err        error
}

func (e *ExecutionError) Error() string {
	if e == nil {
		return ErrInvalidCommandRequest.Error()
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return ErrInvalidCommandRequest.Error()
}

func (e *ExecutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
