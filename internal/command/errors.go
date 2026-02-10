package command

import "errors"

var (
	ErrNotFound              = errors.New("command not found")
	ErrShareNotFound         = errors.New("share not found")
	ErrAssetNotFound         = errors.New("asset not found")
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
