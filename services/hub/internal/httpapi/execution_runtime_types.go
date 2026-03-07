// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import agentcore "goyais/services/hub/internal/agent/core"

// ExecutionUserAnswer captures one answer payload returned by run-control APIs.
type ExecutionUserAnswer struct {
	QuestionID       string
	SelectedOptionID string
	Text             string
}

// executionControlSignal is the internal run-control command envelope.
type executionControlSignal struct {
	Action agentcore.ControlAction
	Answer *ExecutionUserAnswer
}

// pendingUserQuestion stores one pending user-input request for run control.
type pendingUserQuestion struct {
	QuestionID          string
	Question            string
	Options             []map[string]any
	RecommendedOptionID string
	AllowText           bool
	Required            bool
	CallID              string
	ToolName            string
}
