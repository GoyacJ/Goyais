// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package approval provides run-scoped approval and answer waiting primitives.
package approval

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"goyais/services/hub/internal/agent/core"
)

// UserAnswer is the normalized answer payload attached to ControlActionAnswer.
type UserAnswer struct {
	QuestionID       string
	SelectedOptionID string
	Text             string
}

// ControlSignal is one external control event routed to one run.
type ControlSignal struct {
	Action core.ControlAction
	Answer *UserAnswer
}

// Router manages run-local control channels for approval and answer waits.
type Router struct {
	mu     sync.RWMutex
	buffer int
	chans  map[core.RunID]chan ControlSignal
}

// NewRouter constructs a Router with bounded per-run channel buffer.
func NewRouter(buffer int) *Router {
	if buffer <= 0 {
		buffer = 16
	}
	return &Router{
		buffer: buffer,
		chans:  map[core.RunID]chan ControlSignal{},
	}
}

// Register ensures the run has a control channel and returns it.
func (r *Router) Register(runID core.RunID) <-chan ControlSignal {
	normalized := normalizeRunID(runID)
	if normalized == "" {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if current, exists := r.chans[normalized]; exists {
		return current
	}
	created := make(chan ControlSignal, r.buffer)
	r.chans[normalized] = created
	return created
}

// Send pushes one signal to the run control channel.
func (r *Router) Send(runID core.RunID, signal ControlSignal) error {
	control, exists := r.lookup(runID)
	if !exists || control == nil {
		return fmt.Errorf("approval control channel is unavailable for run %q", strings.TrimSpace(string(runID)))
	}
	control <- signal
	return nil
}

// Unregister removes and closes one run control channel.
func (r *Router) Unregister(runID core.RunID) {
	normalized := normalizeRunID(runID)
	if normalized == "" {
		return
	}
	r.mu.Lock()
	control, exists := r.chans[normalized]
	if exists {
		delete(r.chans, normalized)
	}
	r.mu.Unlock()
	if exists && control != nil {
		close(control)
	}
}

// WaitForApproval blocks until one approval action for the run is available.
func (r *Router) WaitForApproval(ctx context.Context, runID core.RunID) (core.ControlAction, error) {
	control, exists := r.lookup(runID)
	if !exists || control == nil {
		return "", fmt.Errorf("approval control channel is unavailable for run %q", strings.TrimSpace(string(runID)))
	}
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case signal, ok := <-control:
			if !ok {
				return "", errors.New("approval control channel is closed")
			}
			switch signal.Action {
			case core.ControlActionApprove, core.ControlActionResume, core.ControlActionDeny, core.ControlActionStop:
				return signal.Action, nil
			default:
				continue
			}
		}
	}
}

// WaitForAnswer blocks until an answer signal for the target question is ready.
func (r *Router) WaitForAnswer(ctx context.Context, runID core.RunID, questionID string) (UserAnswer, error) {
	control, exists := r.lookup(runID)
	if !exists || control == nil {
		return UserAnswer{}, fmt.Errorf("approval control channel is unavailable for run %q", strings.TrimSpace(string(runID)))
	}
	targetQuestionID := strings.TrimSpace(questionID)
	for {
		select {
		case <-ctx.Done():
			return UserAnswer{}, ctx.Err()
		case signal, ok := <-control:
			if !ok {
				return UserAnswer{}, errors.New("approval control channel is closed")
			}
			switch signal.Action {
			case core.ControlActionStop, core.ControlActionDeny:
				return UserAnswer{}, context.Canceled
			case core.ControlActionAnswer:
				if signal.Answer == nil {
					continue
				}
				answer := normalizeAnswer(*signal.Answer)
				if targetQuestionID != "" && answer.QuestionID != targetQuestionID {
					continue
				}
				if answer.SelectedOptionID == "" && answer.Text == "" {
					continue
				}
				return answer, nil
			default:
				continue
			}
		}
	}
}

func (r *Router) lookup(runID core.RunID) (chan ControlSignal, bool) {
	normalized := normalizeRunID(runID)
	if normalized == "" {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	control, exists := r.chans[normalized]
	return control, exists
}

func normalizeRunID(runID core.RunID) core.RunID {
	trimmed := strings.TrimSpace(string(runID))
	if trimmed == "" {
		return ""
	}
	return core.RunID(trimmed)
}

func normalizeAnswer(answer UserAnswer) UserAnswer {
	return UserAnswer{
		QuestionID:       strings.TrimSpace(answer.QuestionID),
		SelectedOptionID: strings.TrimSpace(answer.SelectedOptionID),
		Text:             strings.TrimSpace(answer.Text),
	}
}
