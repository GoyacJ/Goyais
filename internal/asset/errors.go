package asset

import "errors"

var (
	ErrNotFound        = errors.New("asset not found")
	ErrNotImplemented  = errors.New("asset not implemented")
	ErrInvalidRequest  = errors.New("invalid asset request")
	ErrInvalidCursor   = errors.New("invalid cursor")
	ErrForbidden       = errors.New("forbidden")
	ErrObjectStoreFail = errors.New("object store failure")
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
