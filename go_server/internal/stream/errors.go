// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

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
