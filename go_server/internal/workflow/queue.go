// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package workflow

import (
	"encoding/json"
	"strings"
	"time"
)

const (
	queueStatusPending  = "pending"
	queueStatusLeased   = "leased"
	queueStatusDone     = "done"
	queueStatusCanceled = "canceled"
)

type stepQueuePayload struct {
	Mode          string `json:"mode"`
	FailStepKey   string `json:"failStepKey,omitempty"`
	MaxAttempts   int    `json:"maxAttempts,omitempty"`
	BaseBackoffMS int    `json:"baseBackoffMs,omitempty"`
	MaxBackoffMS  int    `json:"maxBackoffMs,omitempty"`
	TestNode      bool   `json:"testNode,omitempty"`
}

type stepQueueItem struct {
	ID          string
	TenantID    string
	WorkspaceID string
	RunID       string
	StepKey     string
	Attempt     int
	Status      string
	AvailableAt time.Time
	LeasedAt    *time.Time
	LeasedBy    string
	PayloadJSON json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func buildStepQueuePayload(mode string, inputs json.RawMessage, defaultFailStep string, testNode bool) stepQueuePayload {
	normalizedMode := strings.TrimSpace(mode)
	if normalizedMode == "" {
		normalizedMode = RunModeSync
	}

	inputMap := decodeJSONMap(inputs)
	retry := parseRetryPolicy(inputMap)
	failStepKey := detectFailStepKey(inputMap)
	if failStepKey == "" {
		failStepKey = strings.TrimSpace(defaultFailStep)
	}

	payload := stepQueuePayload{
		Mode:          normalizedMode,
		MaxAttempts:   1,
		BaseBackoffMS: retry.BaseBackoffMS,
		MaxBackoffMS:  retry.MaxBackoffMS,
		TestNode:      testNode,
	}
	if normalizedMode == RunModeFail {
		payload.FailStepKey = failStepKey
		payload.MaxAttempts = retry.MaxAttempts
		if payload.MaxAttempts <= 0 {
			payload.MaxAttempts = 1
		}
	}
	return payload
}

func decodeStepQueuePayload(raw json.RawMessage) stepQueuePayload {
	payload := stepQueuePayload{
		Mode:        RunModeSync,
		MaxAttempts: 1,
	}
	if len(raw) == 0 {
		return payload
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return payload
	}
	if strings.TrimSpace(payload.Mode) == "" {
		payload.Mode = RunModeSync
	}
	if payload.MaxAttempts <= 0 {
		payload.MaxAttempts = 1
	}
	if payload.BaseBackoffMS < 0 {
		payload.BaseBackoffMS = 0
	}
	if payload.MaxBackoffMS <= 0 {
		payload.MaxBackoffMS = 5000
	}
	return payload
}
