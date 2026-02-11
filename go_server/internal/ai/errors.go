// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

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
