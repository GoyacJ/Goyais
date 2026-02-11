package workflow

import "errors"

var (
	ErrNotImplemented   = errors.New("workflow not implemented")
	ErrInvalidRequest   = errors.New("invalid workflow request")
	ErrInvalidCursor    = errors.New("invalid cursor")
	ErrTemplateNotFound = errors.New("workflow template not found")
	ErrRunNotFound      = errors.New("workflow run not found")
	ErrForbidden        = errors.New("forbidden")
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
