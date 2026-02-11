// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package httpapi

import (
	"net/http"

	"goyais/internal/common/errorx"
)

func NewNotImplementedHandler(messageKey string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", messageKey, nil)
	})
}
