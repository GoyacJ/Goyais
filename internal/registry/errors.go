package registry

import "errors"

var (
	ErrNotImplemented     = errors.New("registry not implemented")
	ErrInvalidRequest     = errors.New("invalid registry request")
	ErrInvalidCursor      = errors.New("invalid cursor")
	ErrCapabilityNotFound = errors.New("capability not found")
	ErrAlgorithmNotFound  = errors.New("algorithm not found")
	ErrForbidden          = errors.New("forbidden")
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
