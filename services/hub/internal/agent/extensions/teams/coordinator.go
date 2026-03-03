// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package teams provides in-process team collaboration primitives for Agent v4.
package teams

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"goyais/services/hub/internal/agent/core"
)

const (
	// StatusPending means task is queued and not actively being worked.
	StatusPending = "pending"
	// StatusInProgress means task is actively executed by a teammate.
	StatusInProgress = "in-progress"
	// StatusCompleted means task was finished and should release file locks.
	StatusCompleted = "completed"
)

const (
	// GateEventTeammateIdle is emitted when teammate becomes idle.
	GateEventTeammateIdle = "TeammateIdle"
	// GateEventTaskCompleted is emitted before task completion is accepted.
	GateEventTaskCompleted = "TaskCompleted"
)

const (
	// PlanStatusPending means the plan awaits lead review.
	PlanStatusPending = "pending"
	// PlanStatusApproved means the plan was accepted by lead.
	PlanStatusApproved = "approved"
	// PlanStatusRejected means the plan was rejected and needs revision.
	PlanStatusRejected = "rejected"
)

var (
	// ErrTaskIDRequired indicates assignment does not include a task ID.
	ErrTaskIDRequired = errors.New("task id is required")
	// ErrMessageTargetRequired indicates message destination is missing.
	ErrMessageTargetRequired = errors.New("message destination agent is required")
	// ErrPlanNotFound indicates review tried to update a missing plan.
	ErrPlanNotFound = errors.New("plan not found")
	// ErrInvalidPlanDecision indicates review decision is unsupported.
	ErrInvalidPlanDecision = errors.New("invalid plan decision")
	// ErrGateRejected indicates gate hook blocked the operation.
	ErrGateRejected = errors.New("team gate rejected operation")
	// ErrFileLockConflict indicates one file is already locked by another task.
	ErrFileLockConflict = errors.New("file lock conflict")
)

// GateDecision describes one gate-hook verdict.
type GateDecision struct {
	Allow    bool
	Feedback string
}

// GateEvaluator evaluates team gate events.
type GateEvaluator interface {
	Evaluate(ctx context.Context, eventName string, payload map[string]any) (GateDecision, error)
}

// GateEvaluatorFunc adapts a function to GateEvaluator.
type GateEvaluatorFunc func(ctx context.Context, eventName string, payload map[string]any) (GateDecision, error)

// Evaluate executes the wrapped function.
func (f GateEvaluatorFunc) Evaluate(ctx context.Context, eventName string, payload map[string]any) (GateDecision, error) {
	return f(ctx, eventName, payload)
}

// Assignment extends core.TeamTask with file-lock metadata.
type Assignment struct {
	Task      core.TeamTask
	FileLocks []string
}

// TaskSnapshot is the read model for one shared task entry.
type TaskSnapshot struct {
	Task      core.TeamTask
	FileLocks []string
	UpdatedAt time.Time
}

// PlanSubmission is one teammate-submitted plan waiting for review.
type PlanSubmission struct {
	ID        string
	FromAgent string
	Title     string
	Content   string
}

// PlanRecord stores review lifecycle state for one plan submission.
type PlanRecord struct {
	ID         string
	FromAgent  string
	Title      string
	Content    string
	Status     string
	Feedback   string
	ReviewedBy string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CoordinatorOptions configures TeamCoordinator behavior.
type CoordinatorOptions struct {
	Now  func() time.Time
	Gate GateEvaluator
}

// Coordinator implements core.TeamCoordinator plus additional team workflows.
type Coordinator struct {
	now  func() time.Time
	gate GateEvaluator

	mu         sync.RWMutex
	sequence   uint64
	tasks      map[string]TaskSnapshot
	fileLocks  map[string]string
	inboxes    map[string][]core.TeamMessage
	plans      map[string]PlanRecord
	messageSeq uint64
}

var _ core.TeamCoordinator = (*Coordinator)(nil)

// NewCoordinator creates an in-memory team coordinator.
func NewCoordinator(options CoordinatorOptions) *Coordinator {
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &Coordinator{
		now:       now,
		gate:      options.Gate,
		tasks:     make(map[string]TaskSnapshot, 16),
		fileLocks: make(map[string]string, 32),
		inboxes:   make(map[string][]core.TeamMessage, 16),
		plans:     make(map[string]PlanRecord, 16),
	}
}

// Assign stores or updates one shared task with optional dependency chain.
func (c *Coordinator) Assign(ctx context.Context, task core.TeamTask) error {
	return c.AssignDetailed(ctx, Assignment{Task: task})
}

// AssignDetailed stores one task entry and applies file-lock constraints.
func (c *Coordinator) AssignDetailed(ctx context.Context, assignment Assignment) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	taskID := strings.TrimSpace(assignment.Task.ID)
	if taskID == "" {
		return ErrTaskIDRequired
	}

	task := assignment.Task
	task.ID = taskID
	task.Status = normalizeStatus(task.Status)
	task.DependsOn = uniqueStrings(task.DependsOn)
	fileLocks := uniqueStrings(assignment.FileLocks)
	updatedAt := c.now().UTC()

	if task.Status == StatusCompleted {
		if err := c.evaluateGate(ctx, GateEventTaskCompleted, map[string]any{
			"taskID":    task.ID,
			"title":     task.Title,
			"status":    task.Status,
			"dependsOn": task.DependsOn,
		}); err != nil {
			return err
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	existing, hasExisting := c.tasks[taskID]
	for _, lockPath := range fileLocks {
		if owner, taken := c.fileLocks[lockPath]; taken && owner != taskID && task.Status != StatusCompleted {
			return fmt.Errorf("%w: %s is owned by task %s", ErrFileLockConflict, lockPath, owner)
		}
	}

	if hasExisting {
		for _, previous := range existing.FileLocks {
			if owner := c.fileLocks[previous]; owner == taskID {
				delete(c.fileLocks, previous)
			}
		}
	}

	if task.Status != StatusCompleted {
		for _, lockPath := range fileLocks {
			c.fileLocks[lockPath] = taskID
		}
	} else {
		fileLocks = nil
	}

	c.tasks[taskID] = TaskSnapshot{
		Task:      task,
		FileLocks: append([]string(nil), fileLocks...),
		UpdatedAt: updatedAt,
	}
	return nil
}

// Tasks returns a sorted snapshot of the shared task list.
func (c *Coordinator) Tasks(ctx context.Context) ([]TaskSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]TaskSnapshot, 0, len(c.tasks))
	for _, record := range c.tasks {
		copyRecord := record
		copyRecord.Task.DependsOn = append([]string(nil), record.Task.DependsOn...)
		copyRecord.FileLocks = append([]string(nil), record.FileLocks...)
		out = append(out, copyRecord)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Task.ID < out[j].Task.ID
	})
	return out, nil
}

// Send enqueues one direct teammate message.
func (c *Coordinator) Send(ctx context.Context, message core.TeamMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	target := strings.TrimSpace(message.ToAgent)
	if target == "" {
		return ErrMessageTargetRequired
	}
	if strings.TrimSpace(message.ID) == "" {
		c.mu.Lock()
		c.messageSeq++
		message.ID = "msg-" + strconv.FormatUint(c.messageSeq, 10)
		c.mu.Unlock()
	}
	if message.SentAt.IsZero() {
		message.SentAt = c.now().UTC()
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.inboxes[target] = append(c.inboxes[target], message)
	return nil
}

// Inbox returns and drains pending messages for one teammate.
func (c *Coordinator) Inbox(ctx context.Context, agentID string) ([]core.TeamMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	target := strings.TrimSpace(agentID)
	if target == "" {
		return []core.TeamMessage{}, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	pending := c.inboxes[target]
	if len(pending) == 0 {
		return []core.TeamMessage{}, nil
	}
	copyPending := append([]core.TeamMessage(nil), pending...)
	delete(c.inboxes, target)
	return copyPending, nil
}

// SubmitPlan registers one teammate plan for lead review.
func (c *Coordinator) SubmitPlan(ctx context.Context, submission PlanSubmission) (PlanRecord, error) {
	if err := ctx.Err(); err != nil {
		return PlanRecord{}, err
	}

	now := c.now().UTC()
	c.mu.Lock()
	defer c.mu.Unlock()

	planID := strings.TrimSpace(submission.ID)
	if planID == "" {
		c.sequence++
		planID = "plan-" + strconv.FormatUint(c.sequence, 10)
	}

	record := PlanRecord{
		ID:        planID,
		FromAgent: strings.TrimSpace(submission.FromAgent),
		Title:     strings.TrimSpace(submission.Title),
		Content:   strings.TrimSpace(submission.Content),
		Status:    PlanStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	c.plans[planID] = record
	return record, nil
}

// ReviewPlan updates one plan with approved/rejected decision and feedback.
func (c *Coordinator) ReviewPlan(ctx context.Context, planID string, decision string, feedback string, reviewer string) (PlanRecord, error) {
	if err := ctx.Err(); err != nil {
		return PlanRecord{}, err
	}
	id := strings.TrimSpace(planID)
	if id == "" {
		return PlanRecord{}, ErrPlanNotFound
	}

	nextStatus := ""
	switch strings.ToLower(strings.TrimSpace(decision)) {
	case PlanStatusApproved:
		nextStatus = PlanStatusApproved
	case PlanStatusRejected:
		nextStatus = PlanStatusRejected
	default:
		return PlanRecord{}, fmt.Errorf("%w: %s", ErrInvalidPlanDecision, decision)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	record, ok := c.plans[id]
	if !ok {
		return PlanRecord{}, fmt.Errorf("%w: %s", ErrPlanNotFound, id)
	}

	record.Status = nextStatus
	record.Feedback = strings.TrimSpace(feedback)
	record.ReviewedBy = strings.TrimSpace(reviewer)
	record.UpdatedAt = c.now().UTC()
	c.plans[id] = record
	return record, nil
}

// NotifyTeammateIdle emits teammate idle gate event.
func (c *Coordinator) NotifyTeammateIdle(ctx context.Context, agentID string) error {
	return c.evaluateGate(ctx, GateEventTeammateIdle, map[string]any{
		"agentID": strings.TrimSpace(agentID),
	})
}

func (c *Coordinator) evaluateGate(ctx context.Context, eventName string, payload map[string]any) error {
	if c.gate == nil {
		return nil
	}
	decision, err := c.gate.Evaluate(ctx, eventName, payload)
	if err != nil {
		return err
	}
	if decision.Allow {
		return nil
	}
	reason := strings.TrimSpace(decision.Feedback)
	if reason == "" {
		reason = "gate policy rejected " + eventName
	}
	return fmt.Errorf("%w: %s", ErrGateRejected, reason)
}

func normalizeStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case StatusInProgress:
		return StatusInProgress
	case StatusCompleted:
		return StatusCompleted
	default:
		return StatusPending
	}
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}
