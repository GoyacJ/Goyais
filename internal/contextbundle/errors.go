package contextbundle

import "errors"

var (
	ErrInvalidRequest = errors.New("invalid context bundle request")
	ErrInvalidCursor  = errors.New("invalid cursor")
	ErrNotFound       = errors.New("context bundle not found")
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
