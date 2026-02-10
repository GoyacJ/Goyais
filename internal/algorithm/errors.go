package algorithm

import "errors"

var (
	ErrNotImplemented    = errors.New("algorithm not implemented")
	ErrInvalidRequest    = errors.New("invalid algorithm request")
	ErrAlgorithmNotFound = errors.New("algorithm not found")
	ErrForbidden         = errors.New("forbidden")
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
