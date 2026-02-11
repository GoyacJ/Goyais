// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"goyais/internal/command"
	"goyais/internal/common/errorx"
)

func (h *apiHandler) handleAlgorithmRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	route := strings.TrimPrefix(r.URL.Path, "/api/v1/algorithms/")
	if strings.TrimSpace(route) == "" || strings.Contains(route, "/") || !strings.HasSuffix(route, ":run") {
		errorx.Write(w, http.StatusNotFound, "ALGORITHM_NOT_FOUND", "error.algorithm.not_found", map[string]any{"path": r.URL.Path})
		return
	}

	algorithmID := strings.TrimSpace(strings.TrimSuffix(route, ":run"))
	if algorithmID == "" || strings.Contains(algorithmID, ":") {
		errorx.Write(w, http.StatusNotFound, "ALGORITHM_NOT_FOUND", "error.algorithm.not_found", map[string]any{"path": r.URL.Path})
		return
	}
	h.handleRunAlgorithm(w, r, algorithmID)
}

func (h *apiHandler) handleRunAlgorithm(w http.ResponseWriter, r *http.Request, algorithmID string) {
	if h.commandService == nil {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.algorithm.not_implemented", nil)
		return
	}

	reqCtx, ok := requireRequestContext(w, r)
	if !ok {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", nil)
		return
	}
	if len(body) == 0 {
		body = []byte("{}")
	}

	var req struct {
		Inputs     json.RawMessage `json:"inputs"`
		Visibility string          `json:"visibility"`
		Mode       string          `json:"mode"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		errorx.Write(w, http.StatusBadRequest, "INVALID_JSON", "error.request.invalid_json", nil)
		return
	}
	if len(req.Inputs) == 0 {
		req.Inputs = json.RawMessage(`{}`)
	}

	payload, _ := json.Marshal(map[string]any{
		"algorithmId": algorithmID,
		"inputs":      req.Inputs,
		"visibility":  strings.TrimSpace(req.Visibility),
		"mode":        strings.TrimSpace(req.Mode),
	})
	cmd, err := h.commandService.Submit(
		r.Context(),
		reqCtx,
		"algorithm.run",
		payload,
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		strings.TrimSpace(req.Visibility),
	)
	if err != nil {
		writeAlgorithmCommandError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"resource":   readAlgorithmResourceFromResult(cmd.Result),
		"commandRef": toCommandRefPayload(cmd),
	})
}

func readAlgorithmResourceFromResult(resultRaw json.RawMessage) map[string]any {
	resource := map[string]any{}
	if len(resultRaw) > 0 {
		var parsed map[string]any
		if json.Unmarshal(resultRaw, &parsed) == nil {
			if candidate, ok := parsed["run"].(map[string]any); ok {
				resource = candidate
			}
		}
	}
	if len(resource) == 0 {
		resource = map[string]any{"id": "", "status": command.StatusSucceeded}
	}
	return resource
}

func writeAlgorithmCommandError(w http.ResponseWriter, err error) {
	var execErr *command.ExecutionError
	if errors.As(err, &execErr) && strings.TrimSpace(execErr.Code) != "" && strings.TrimSpace(execErr.MessageKey) != "" {
		status := http.StatusInternalServerError
		switch strings.ToUpper(strings.TrimSpace(execErr.Code)) {
		case "INVALID_ALGORITHM_REQUEST":
			status = http.StatusBadRequest
		case "ALGORITHM_NOT_FOUND":
			status = http.StatusNotFound
		case "NOT_IMPLEMENTED":
			status = http.StatusNotImplemented
		case "FORBIDDEN":
			status = http.StatusForbidden
		}
		errorx.Write(w, status, strings.ToUpper(strings.TrimSpace(execErr.Code)), strings.TrimSpace(execErr.MessageKey), nil)
		return
	}

	switch {
	case errors.Is(err, command.ErrInvalidCommandRequest):
		errorx.Write(w, http.StatusBadRequest, "INVALID_ALGORITHM_REQUEST", "error.algorithm.invalid_request", nil)
	case errors.Is(err, command.ErrNotImplemented):
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "error.algorithm.not_implemented", nil)
	case errors.Is(err, command.ErrForbidden):
		details := map[string]any{}
		var forbiddenErr *command.ForbiddenError
		if errors.As(err, &forbiddenErr) && forbiddenErr.Reason != "" {
			details["reason"] = forbiddenErr.Reason
		}
		errorx.Write(w, http.StatusForbidden, "FORBIDDEN", "error.authz.forbidden", details)
	default:
		errorx.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error.common.internal", map[string]any{"reason": err.Error()})
	}
}
