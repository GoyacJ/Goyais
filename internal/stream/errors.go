package stream

import "errors"

var (
	ErrNotImplemented = errors.New("stream not implemented")
	ErrInvalidRequest = errors.New("invalid stream request")
	ErrInvalidCursor  = errors.New("invalid cursor")
	ErrStreamNotFound = errors.New("stream not found")
	ErrRecordingNotFound = errors.New("stream recording not found")
	ErrForbidden      = errors.New("forbidden")
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
