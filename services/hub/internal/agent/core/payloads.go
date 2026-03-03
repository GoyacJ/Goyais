// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

// OutputDeltaPayload carries incremental model output chunks.
type OutputDeltaPayload struct {
	Delta     string
	ToolUseID string
}

func (OutputDeltaPayload) isEventPayload() {}

// ApprovalNeededPayload describes a permission checkpoint before tool use.
type ApprovalNeededPayload struct {
	ToolName  string
	Input     map[string]any
	RiskLevel string
}

func (ApprovalNeededPayload) isEventPayload() {}

// RunFailedPayload describes a terminal failure with structured metadata.
type RunFailedPayload struct {
	Code     string
	Message  string
	Metadata map[string]any
}

func (RunFailedPayload) isEventPayload() {}

// RunCompletedPayload summarizes completion metadata for a successful run.
type RunCompletedPayload struct {
	UsageTokens int
}

func (RunCompletedPayload) isEventPayload() {}

// RunCancelledPayload captures who/what cancelled the run.
type RunCancelledPayload struct {
	Reason string
}

func (RunCancelledPayload) isEventPayload() {}
