// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Workflow capability execution semantics builders.

package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

func buildStepExecutionOutput(
	stepKey string,
	stepType string,
	mode string,
	status string,
	attempt int,
	inputs map[string]any,
	errorCode string,
	messageKey string,
	reason string,
	willRetry bool,
	retryAfterMS int,
) json.RawMessage {
	normalizedType := strings.TrimSpace(stepType)
	if normalizedType == "" {
		normalizedType = "noop"
	}
	normalizedStatus := strings.TrimSpace(status)
	if normalizedStatus == "" {
		normalizedStatus = StepStatusPending
	}
	if attempt <= 0 {
		attempt = 1
	}

	payload := map[string]any{
		"step": map[string]any{
			"key":     strings.TrimSpace(stepKey),
			"type":    normalizedType,
			"status":  normalizedStatus,
			"attempt": attempt,
		},
		"capability": map[string]any{
			"executor": capabilityExecutorForStepType(normalizedType),
			"contract": "capability.v1",
		},
		"input": map[string]any{
			"mode":   normalizeExecutionMode(mode),
			"params": cloneAnyMap(inputs),
		},
	}

	switch normalizedStatus {
	case StepStatusSucceeded:
		payload["output"] = map[string]any{
			"result": map[string]any{
				"status":  "ok",
				"summary": fmt.Sprintf("capability %s completed", normalizedType),
			},
			"artifacts": []map[string]any{{
				"kind": "log",
				"ref":  fmt.Sprintf("log://workflow-step/%s", strings.TrimSpace(stepKey)),
			}},
			"metrics": map[string]any{
				"attempt": attempt,
			},
		}
	case StepStatusRunning:
		payload["output"] = map[string]any{
			"progress": map[string]any{
				"state":   "executing",
				"percent": 0,
			},
		}
	case StepStatusPending:
		payload["output"] = map[string]any{
			"progress": map[string]any{
				"state": "queued",
			},
		}
	case StepStatusSkipped:
		payload["output"] = map[string]any{
			"result": map[string]any{
				"status": "skipped",
				"reason": strings.TrimSpace(reason),
			},
		}
	case StepStatusFailed:
		payload["output"] = map[string]any{
			"result": map[string]any{
				"status":      "error",
				"recoverable": willRetry,
			},
		}
	case StepStatusCanceled:
		payload["output"] = map[string]any{
			"result": map[string]any{
				"status": "canceled",
			},
		}
	default:
		payload["output"] = map[string]any{}
	}

	errorPayload := map[string]any{}
	if strings.TrimSpace(errorCode) != "" {
		errorPayload["code"] = strings.TrimSpace(errorCode)
	}
	if strings.TrimSpace(messageKey) != "" {
		errorPayload["messageKey"] = strings.TrimSpace(messageKey)
	}
	if strings.TrimSpace(reason) != "" {
		errorPayload["reason"] = strings.TrimSpace(reason)
	}
	if willRetry {
		recovery := map[string]any{"willRetry": true}
		if retryAfterMS > 0 {
			recovery["retryAfterMs"] = retryAfterMS
		}
		errorPayload["recovery"] = recovery
	}
	if len(errorPayload) > 0 {
		payload["error"] = errorPayload
	}

	return mustJSONObjectRaw(payload)
}

func buildStepExecutionOutputFromRawInput(
	stepKey string,
	stepType string,
	mode string,
	status string,
	attempt int,
	inputsRaw json.RawMessage,
	errorCode string,
	messageKey string,
	reason string,
	willRetry bool,
	retryAfterMS int,
) json.RawMessage {
	return buildStepExecutionOutput(
		stepKey,
		stepType,
		mode,
		status,
		attempt,
		decodeJSONMap(inputsRaw),
		errorCode,
		messageKey,
		reason,
		willRetry,
		retryAfterMS,
	)
}

func buildRunExecutionOutputFromSteps(mode string, runStatus string, steps []plannedStep, extra map[string]any) json.RawMessage {
	stepKeys := make([]string, 0, len(steps))
	statusCounts := map[string]int{
		StepStatusPending:   0,
		StepStatusRunning:   0,
		StepStatusSucceeded: 0,
		StepStatusFailed:    0,
		StepStatusSkipped:   0,
		StepStatusCanceled:  0,
	}
	for _, step := range steps {
		key := strings.TrimSpace(step.Key)
		if key != "" {
			stepKeys = append(stepKeys, key)
		}
		if _, ok := statusCounts[step.Status]; !ok {
			statusCounts[step.Status] = 0
		}
		statusCounts[step.Status]++
	}
	return buildRunExecutionOutputFromCounts(mode, runStatus, stepKeys, statusCounts, extra)
}

func buildRunExecutionOutputFromCounts(
	mode string,
	runStatus string,
	stepKeys []string,
	statusCounts map[string]int,
	extra map[string]any,
) json.RawMessage {
	normalizedStatusCounts := map[string]int{
		StepStatusPending:   0,
		StepStatusRunning:   0,
		StepStatusSucceeded: 0,
		StepStatusFailed:    0,
		StepStatusSkipped:   0,
		StepStatusCanceled:  0,
	}
	for key, value := range statusCounts {
		normalizedStatusCounts[key] = value
	}

	totalSteps := len(stepKeys)
	if totalSteps == 0 {
		totalSteps = normalizedStatusCounts[StepStatusPending] +
			normalizedStatusCounts[StepStatusRunning] +
			normalizedStatusCounts[StepStatusSucceeded] +
			normalizedStatusCounts[StepStatusFailed] +
			normalizedStatusCounts[StepStatusSkipped] +
			normalizedStatusCounts[StepStatusCanceled]
	}

	runPayload := map[string]any{
		"status":             strings.TrimSpace(runStatus),
		"mode":               normalizeExecutionMode(mode),
		"executionSemantics": "capability.v1",
	}
	for key, value := range extra {
		runPayload[key] = value
	}

	return mustJSONObjectRaw(map[string]any{
		"run": runPayload,
		"capabilitySummary": map[string]any{
			"stepKeys":   append([]string{}, stepKeys...),
			"totalSteps": totalSteps,
			"statusCounts": map[string]any{
				StepStatusPending:   normalizedStatusCounts[StepStatusPending],
				StepStatusRunning:   normalizedStatusCounts[StepStatusRunning],
				StepStatusSucceeded: normalizedStatusCounts[StepStatusSucceeded],
				StepStatusFailed:    normalizedStatusCounts[StepStatusFailed],
				StepStatusSkipped:   normalizedStatusCounts[StepStatusSkipped],
				StepStatusCanceled:  normalizedStatusCounts[StepStatusCanceled],
			},
		},
	})
}

func normalizeExecutionMode(mode string) string {
	normalized := strings.TrimSpace(mode)
	if normalized == "" {
		return RunModeSync
	}
	return normalized
}

func capabilityExecutorForStepType(stepType string) string {
	normalized := strings.ToLower(strings.TrimSpace(stepType))
	switch {
	case strings.HasPrefix(normalized, "input"):
		return "input.gateway"
	case strings.HasPrefix(normalized, "tool"):
		return "tool.dispatcher"
	case strings.HasPrefix(normalized, "model"):
		return "model.gateway"
	case strings.HasPrefix(normalized, "algorithm"):
		return "algorithm.runner"
	case strings.HasPrefix(normalized, "transform"):
		return "transform.pipeline"
	case strings.HasPrefix(normalized, "control"):
		return "control.flow"
	case strings.HasPrefix(normalized, "output"):
		return "output.writer"
	default:
		return "capability.runner"
	}
}

func cloneAnyMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}
	return cloned
}
