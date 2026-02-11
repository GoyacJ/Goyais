package ai

import "errors"

var (
	ErrNotImplemented  = errors.New("ai not implemented")
	ErrInvalidRequest  = errors.New("invalid ai request")
	ErrInvalidCursor   = errors.New("invalid cursor")
	ErrSessionNotFound = errors.New("ai session not found")
	ErrForbidden       = errors.New("forbidden")
)

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
