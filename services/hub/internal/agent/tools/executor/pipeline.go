// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package executor contains the Agent v4 tool execution pipeline.
package executor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"goyais/services/hub/internal/agent/core"
	"goyais/services/hub/internal/agent/tools/interaction"
	"goyais/services/hub/internal/agent/tools/spec"
)

// ToolCall is one normalized tool invocation request.
type ToolCall struct {
	CallID        string
	Name          string
	Input         map[string]any
	ArgumentError string
}

// ToolContext carries execution environment metadata for tools.
type ToolContext struct {
	WorkingDir string
	Env        map[string]string
}

// RunRequest is the executor backend input for one tool attempt.
type RunRequest struct {
	SessionMode string
	SafeMode    bool
	Approved    bool
	ToolContext ToolContext
	Call        ToolCall
}

// Runner is the side-effect boundary for tool execution.
type Runner interface {
	Execute(ctx context.Context, req RunRequest) (map[string]any, error)
}

// ApprovalRequest contains details required to ask for tool approval.
type ApprovalRequest struct {
	ToolName string
	CallID   string
	Reason   string
}

// ApprovalAction is the user decision for one approval checkpoint.
type ApprovalAction string

const (
	ApprovalActionApprove ApprovalAction = "approve"
	ApprovalActionResume  ApprovalAction = "resume"
	ApprovalActionDeny    ApprovalAction = "deny"
	ApprovalActionStop    ApprovalAction = "stop"
)

// ApprovalWaiter blocks until approval action is available.
type ApprovalWaiter interface {
	WaitForApproval(ctx context.Context, req ApprovalRequest) (ApprovalAction, error)
}

// UserAnswer is the normalized user answer for a tool-generated question.
type UserAnswer struct {
	QuestionID       string
	SelectedOptionID string
	Text             string
}

// UserAnswerWaiter blocks until a user answer is available.
type UserAnswerWaiter interface {
	WaitForAnswer(ctx context.Context, question interaction.PendingUserQuestion) (UserAnswer, error)
}

// Dependencies declares explicit collaborators for the pipeline.
type Dependencies struct {
	Runner           Runner
	Specs            spec.Resolver
	ApprovalWaiter   ApprovalWaiter
	UserAnswerWaiter UserAnswerWaiter
}

// Pipeline executes tool calls with concurrency grouping and approval retries.
type Pipeline struct {
	runner           Runner
	specs            spec.Resolver
	approvalWaiter   ApprovalWaiter
	userAnswerWaiter UserAnswerWaiter
}

var _ core.ToolExecutor = (*Pipeline)(nil)

// NewPipeline wires one executor pipeline from explicit dependencies.
func NewPipeline(deps Dependencies) *Pipeline {
	return &Pipeline{
		runner:           deps.Runner,
		specs:            deps.Specs,
		approvalWaiter:   deps.ApprovalWaiter,
		userAnswerWaiter: deps.UserAnswerWaiter,
	}
}

// Execute adapts Pipeline to core.ToolExecutor.
//
// The core contract intentionally only covers a single tool call with stable
// tool output and optional structured error; approval/interaction details remain
// inside package-local ExecuteSingle/ExecuteBatch APIs.
func (p *Pipeline) Execute(ctx context.Context, call core.ToolCall) (core.ToolResult, error) {
	if err := call.Validate(); err != nil {
		return core.ToolResult{}, err
	}
	item, err := p.ExecuteSingle(ctx, ExecuteSingleRequest{
		Call: ToolCall{
			CallID:        string(call.RunID) + ":" + string(call.SessionID),
			Name:          call.ToolName,
			Input:         cloneMapAny(call.Input),
			ArgumentError: "",
		},
	})
	if err != nil {
		return core.ToolResult{}, err
	}
	out := core.ToolResult{
		ToolName: item.ToolName,
		Output:   cloneMapAny(item.Output),
	}
	if item.ErrorText != "" {
		out.Error = &core.RunError{
			Code:    "tool_execution_failed",
			Message: item.ErrorText,
		}
	}
	return out, nil
}

// ExecuteSingleRequest is the runtime input for one tool call.
type ExecuteSingleRequest struct {
	Call        ToolCall
	SessionMode string
	SafeMode    bool
	ToolContext ToolContext
}

// ExecuteBatchRequest is the runtime input for one ordered tool-call batch.
type ExecuteBatchRequest struct {
	Calls       []ToolCall
	SessionMode string
	SafeMode    bool
	ToolContext ToolContext
}

// ExecuteSingleResult is the normalized result for one tool call.
type ExecuteSingleResult struct {
	CallID          string
	ToolName        string
	Output          map[string]any
	OutputText      string
	ErrorText       string
	PendingQuestion *interaction.PendingUserQuestion
}

// OK reports whether the result is successful.
func (r ExecuteSingleResult) OK() bool {
	return strings.TrimSpace(r.ErrorText) == ""
}

// ApprovalRequiredError asks caller for explicit approval before re-try.
type ApprovalRequiredError struct {
	ToolName string
	Reason   string
}

func (e *ApprovalRequiredError) Error() string {
	toolName := strings.TrimSpace(e.ToolName)
	reason := strings.TrimSpace(e.Reason)
	if toolName == "" {
		toolName = "tool"
	}
	if reason == "" {
		return toolName + " requires approval"
	}
	return toolName + " requires approval: " + reason
}

// ExecuteBatch preserves call order while fan-out running concurrency-safe
// call groups.
func (p *Pipeline) ExecuteBatch(ctx context.Context, req ExecuteBatchRequest) ([]ExecuteSingleResult, error) {
	if p == nil || p.runner == nil {
		return nil, errors.New("tool runner is not configured")
	}
	if len(req.Calls) == 0 {
		return nil, nil
	}
	results := make([]ExecuteSingleResult, len(req.Calls))
	for index := 0; index < len(req.Calls); {
		if !p.canRunInParallel(req.Calls[index].Name) {
			item, err := p.ExecuteSingle(ctx, ExecuteSingleRequest{
				Call:        req.Calls[index],
				SessionMode: req.SessionMode,
				SafeMode:    req.SafeMode,
				ToolContext: req.ToolContext,
			})
			if err != nil {
				return nil, err
			}
			results[index] = item
			index++
			continue
		}

		groupEnd := index
		for groupEnd < len(req.Calls) && p.canRunInParallel(req.Calls[groupEnd].Name) {
			groupEnd++
		}

		groupErr := make(chan error, groupEnd-index)
		var wg sync.WaitGroup
		for i := index; i < groupEnd; i++ {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				item, err := p.ExecuteSingle(ctx, ExecuteSingleRequest{
					Call:        req.Calls[i],
					SessionMode: req.SessionMode,
					SafeMode:    req.SafeMode,
					ToolContext: req.ToolContext,
				})
				if err != nil {
					groupErr <- err
					return
				}
				results[i] = item
			}()
		}
		wg.Wait()
		close(groupErr)

		for err := range groupErr {
			if err != nil {
				return nil, err
			}
		}
		index = groupEnd
	}
	return results, nil
}

// ExecuteSingle runs one tool call with approval retry and optional user-answer
// capture from tool output.
func (p *Pipeline) ExecuteSingle(ctx context.Context, req ExecuteSingleRequest) (ExecuteSingleResult, error) {
	if p == nil || p.runner == nil {
		return ExecuteSingleResult{}, errors.New("tool runner is not configured")
	}

	call := normalizeCall(req.Call)
	if call.CallID == "" {
		call.CallID = "call_" + randomHex(6)
	}
	if call.Name == "" {
		return ExecuteSingleResult{
			CallID:    call.CallID,
			ToolName:  "unknown",
			ErrorText: "tool call is missing function name",
		}, nil
	}
	if call.ArgumentError != "" {
		return ExecuteSingleResult{
			CallID:    call.CallID,
			ToolName:  call.Name,
			ErrorText: "invalid tool arguments: " + call.ArgumentError,
		}, nil
	}

	approved := false
	for {
		output, err := p.runner.Execute(ctx, RunRequest{
			SessionMode: req.SessionMode,
			SafeMode:    req.SafeMode,
			Approved:    approved,
			ToolContext: req.ToolContext,
			Call:        call,
		})
		if err == nil {
			return p.resolveOutput(ctx, call, output)
		}

		var approvalErr *ApprovalRequiredError
		if errors.As(err, &approvalErr) {
			if p.approvalWaiter == nil {
				return ExecuteSingleResult{}, err
			}
			action, waitErr := p.approvalWaiter.WaitForApproval(ctx, ApprovalRequest{
				ToolName: call.Name,
				CallID:   call.CallID,
				Reason:   strings.TrimSpace(approvalErr.Reason),
			})
			if waitErr != nil {
				return ExecuteSingleResult{}, waitErr
			}
			switch action {
			case ApprovalActionStop:
				return ExecuteSingleResult{}, context.Canceled
			case ApprovalActionDeny:
				errText := strings.TrimSpace(approvalErr.Reason)
				if errText == "" {
					errText = "tool call denied by user"
				}
				return ExecuteSingleResult{
					CallID:    call.CallID,
					ToolName:  call.Name,
					ErrorText: errText,
				}, nil
			case ApprovalActionApprove, ApprovalActionResume:
				approved = true
				continue
			default:
				continue
			}
		}

		errText := strings.TrimSpace(err.Error())
		if errText == "" {
			errText = "tool execution failed"
		}
		return ExecuteSingleResult{
			CallID:    call.CallID,
			ToolName:  call.Name,
			ErrorText: errText,
		}, nil
	}
}

func (p *Pipeline) canRunInParallel(name string) bool {
	if p.specs == nil {
		return false
	}
	item, exists := p.specs.Lookup(strings.TrimSpace(name))
	if !exists {
		return false
	}
	return item.ConcurrencySafe && !item.NeedsPermissions
}

func (p *Pipeline) resolveOutput(ctx context.Context, call ToolCall, output map[string]any) (ExecuteSingleResult, error) {
	result := ExecuteSingleResult{
		CallID:     call.CallID,
		ToolName:   call.Name,
		Output:     cloneMapAny(output),
		OutputText: renderOutput(output),
	}
	if !interaction.RequiresUserInputFromToolResult(output) {
		return result, nil
	}

	question := interaction.NormalizePendingUserQuestion(output, call.CallID, call.Name)
	result.PendingQuestion = &question
	if p.userAnswerWaiter == nil {
		return result, nil
	}
	answer, waitErr := p.userAnswerWaiter.WaitForAnswer(ctx, question)
	if waitErr != nil {
		return ExecuteSingleResult{}, waitErr
	}
	mergedOutput := cloneMapAny(output)
	mergedOutput["requires_user_input"] = false
	mergedOutput["answer"] = map[string]any{
		"question_id":        strings.TrimSpace(answer.QuestionID),
		"selected_option_id": strings.TrimSpace(answer.SelectedOptionID),
		"text":               strings.TrimSpace(answer.Text),
	}
	result.Output = mergedOutput
	result.OutputText = renderOutput(mergedOutput)
	return result, nil
}

func normalizeCall(call ToolCall) ToolCall {
	call.CallID = strings.TrimSpace(call.CallID)
	call.Name = strings.TrimSpace(call.Name)
	call.ArgumentError = strings.TrimSpace(call.ArgumentError)
	if call.Input == nil {
		call.Input = map[string]any{}
	}
	return call
}

func cloneMapAny(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func renderOutput(output map[string]any) string {
	if len(output) == 0 {
		return "{}"
	}
	payload, err := json.Marshal(output)
	if err != nil {
		return fmt.Sprintf("%v", output)
	}
	return string(payload)
}

func randomHex(bytesLen int) string {
	if bytesLen <= 0 {
		return ""
	}
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "fallback"
	}
	return strings.ToLower(hex.EncodeToString(buf))
}
