// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"goyais/services/hub/internal/agent/core"
	runtimesession "goyais/services/hub/internal/agent/runtime/session"
)

type engineStub struct {
	startReq core.StartSessionRequest
	submit   struct {
		sessionID string
		input     core.UserInput
	}
	control struct {
		req core.ControlRequest
	}
	subscribe struct {
		sessionID string
		cursor    string
	}
	startHandle core.SessionHandle
	runID       string
	startErr    error
	submitErr   error
	controlErr  error
	subErr      error
	sub         core.EventSubscription
}

func (s *engineStub) StartSession(_ context.Context, req core.StartSessionRequest) (core.SessionHandle, error) {
	s.startReq = req
	if s.startErr != nil {
		return core.SessionHandle{}, s.startErr
	}
	if s.startHandle.CreatedAt.IsZero() {
		s.startHandle.CreatedAt = time.Date(2026, 3, 3, 9, 0, 0, 0, time.UTC)
	}
	if s.startHandle.SessionID == "" {
		s.startHandle.SessionID = core.SessionID("sess_default")
	}
	return s.startHandle, nil
}

func (s *engineStub) Submit(_ context.Context, sessionID string, input core.UserInput) (string, error) {
	s.submit.sessionID = sessionID
	s.submit.input = input
	if s.submitErr != nil {
		return "", s.submitErr
	}
	if s.runID == "" {
		return "run_default", nil
	}
	return s.runID, nil
}

func (s *engineStub) Control(_ context.Context, req core.ControlRequest) error {
	s.control.req = req
	return s.controlErr
}

func (s *engineStub) Subscribe(_ context.Context, sessionID string, cursor string) (core.EventSubscription, error) {
	s.subscribe.sessionID = sessionID
	s.subscribe.cursor = cursor
	if s.subErr != nil {
		return nil, s.subErr
	}
	if s.sub == nil {
		return &subscriptionStub{events: make(chan core.EventEnvelope)}, nil
	}
	return s.sub, nil
}

type subscriptionStub struct {
	events chan core.EventEnvelope
	once   sync.Once
}

func (s *subscriptionStub) Events() <-chan core.EventEnvelope { return s.events }

func (s *subscriptionStub) Close() error {
	s.once.Do(func() {
		close(s.events)
	})
	return nil
}

type lifecycleStub struct {
	resumeReq  runtimesession.ResumeRequest
	forkReq    runtimesession.ForkRequest
	rewindReq  runtimesession.RewindRequest
	clearReq   runtimesession.ClearRequest
	handoffReq runtimesession.HandoffRequest

	state    runtimesession.State
	snapshot runtimesession.HandoffSnapshot

	resumeErr  error
	forkErr    error
	rewindErr  error
	clearErr   error
	handoffErr error
}

type checkpointServiceStub struct {
	req  SessionCheckpointRollbackRequest
	resp SessionStateResponse
	err  error
}

func (s *checkpointServiceStub) RollbackToCheckpoint(_ context.Context, req SessionCheckpointRollbackRequest) (SessionStateResponse, error) {
	s.req = req
	if s.err != nil {
		return SessionStateResponse{}, s.err
	}
	return s.resp, nil
}

func (s *lifecycleStub) Resume(_ context.Context, req runtimesession.ResumeRequest) (runtimesession.State, error) {
	s.resumeReq = req
	if s.resumeErr != nil {
		return runtimesession.State{}, s.resumeErr
	}
	return s.state, nil
}

func (s *lifecycleStub) Fork(_ context.Context, req runtimesession.ForkRequest) (runtimesession.State, error) {
	s.forkReq = req
	if s.forkErr != nil {
		return runtimesession.State{}, s.forkErr
	}
	return s.state, nil
}

func (s *lifecycleStub) Rewind(_ context.Context, req runtimesession.RewindRequest) (runtimesession.State, error) {
	s.rewindReq = req
	if s.rewindErr != nil {
		return runtimesession.State{}, s.rewindErr
	}
	return s.state, nil
}

func (s *lifecycleStub) Clear(_ context.Context, req runtimesession.ClearRequest) (runtimesession.State, error) {
	s.clearReq = req
	if s.clearErr != nil {
		return runtimesession.State{}, s.clearErr
	}
	return s.state, nil
}

func (s *lifecycleStub) Handoff(_ context.Context, req runtimesession.HandoffRequest) (runtimesession.HandoffSnapshot, error) {
	s.handoffReq = req
	if s.handoffErr != nil {
		return runtimesession.HandoffSnapshot{}, s.handoffErr
	}
	return s.snapshot, nil
}

func TestServiceStartSessionDelegatesToEngine(t *testing.T) {
	engine := &engineStub{startHandle: core.SessionHandle{SessionID: core.SessionID("sess_42"), CreatedAt: time.Date(2026, 3, 3, 9, 1, 0, 0, time.UTC)}}
	svc := NewService(engine)

	resp, err := svc.StartSession(context.Background(), StartSessionRequest{
		WorkingDir:            "/tmp/work",
		AdditionalDirectories: []string{"/tmp/a", "/tmp/b"},
	})
	if err != nil {
		t.Fatalf("start session failed: %v", err)
	}
	if resp.SessionID != "sess_42" {
		t.Fatalf("session id = %q, want %q", resp.SessionID, "sess_42")
	}
	if resp.CreatedAt != "2026-03-03T09:01:00Z" {
		t.Fatalf("created_at = %q", resp.CreatedAt)
	}
	if engine.startReq.WorkingDir != "/tmp/work" {
		t.Fatalf("engine start working dir = %q", engine.startReq.WorkingDir)
	}
}

func TestServiceRewindSessionUsesCheckpointServiceWhenConfigured(t *testing.T) {
	engine := &engineStub{}
	lifecycle := &lifecycleStub{
		state: runtimesession.State{
			SessionID:        core.SessionID("sess_lifecycle"),
			LastCheckpointID: core.CheckpointID("cp_lifecycle"),
			NextCursor:       1,
		},
	}
	checkpoints := &checkpointServiceStub{
		resp: SessionStateResponse{
			SessionID:        "sess_checkpoint",
			LastCheckpointID: "cp_checkpoint",
			NextCursor:       5,
			PermissionMode:   "plan",
		},
	}
	svc := NewServiceWithLifecycleAndCheckpoints(engine, lifecycle, checkpoints)

	rewound, err := svc.RewindSession(context.Background(), RewindSessionRequest{
		SessionID:            "sess_checkpoint",
		CheckpointID:         "cp_checkpoint",
		TargetCursor:         5,
		ClearTempPermissions: true,
	})
	if err != nil {
		t.Fatalf("rewind session failed: %v", err)
	}
	if checkpoints.req.SessionID != "sess_checkpoint" || checkpoints.req.CheckpointID != "cp_checkpoint" {
		t.Fatalf("unexpected checkpoint request %#v", checkpoints.req)
	}
	if checkpoints.req.TargetCursor != 5 || !checkpoints.req.ClearTempPermissions {
		t.Fatalf("unexpected checkpoint request %#v", checkpoints.req)
	}
	if lifecycle.rewindReq.SessionID != "" {
		t.Fatalf("expected lifecycle rewind to be bypassed, got %#v", lifecycle.rewindReq)
	}
	if rewound.LastCheckpointID != "cp_checkpoint" || rewound.NextCursor != 5 || rewound.PermissionMode != "plan" {
		t.Fatalf("unexpected rewind response %#v", rewound)
	}
}

func TestServiceSubmitDelegatesToEngine(t *testing.T) {
	engine := &engineStub{runID: "run_99"}
	svc := NewService(engine)
	resp, err := svc.Submit(context.Background(), SubmitRequest{
		SessionID: "sess_1",
		Input:     "hello",
		Metadata: map[string]string{
			"source": "http",
		},
	})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	if resp.RunID != "run_99" {
		t.Fatalf("run id = %q, want run_99", resp.RunID)
	}
	if engine.submit.sessionID != "sess_1" || engine.submit.input.Text != "hello" {
		t.Fatalf("unexpected submit delegation: %#v", engine.submit)
	}
	if engine.submit.input.Metadata["source"] != "http" {
		t.Fatalf("metadata source = %q", engine.submit.input.Metadata["source"])
	}
}

func TestServiceSubmitForwardsExplicitRuntimeConfig(t *testing.T) {
	engine := &engineStub{runID: "run_typed"}
	svc := NewService(engine)
	runtimeConfig := &core.RuntimeConfig{
		Model: core.RuntimeModelConfig{
			ProviderName:  "openai-compatible",
			Endpoint:      "https://example.invalid/v1",
			ModelName:     "gpt-test",
			MaxModelTurns: 6,
		},
		Tooling: core.RuntimeToolingConfig{
			PermissionMode: core.PermissionModePlan,
		},
	}

	resp, err := svc.Submit(context.Background(), SubmitRequest{
		SessionID:     "sess_2",
		Input:         "hello",
		Metadata:      map[string]string{"source": "http"},
		RuntimeConfig: runtimeConfig,
	})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	if resp.RunID != "run_typed" {
		t.Fatalf("run id = %q, want run_typed", resp.RunID)
	}
	if engine.submit.input.RuntimeConfig == nil {
		t.Fatal("expected typed runtime config to be populated")
	}
	if engine.submit.input.RuntimeConfig.Model.ProviderName != "openai-compatible" {
		t.Fatalf("provider = %q", engine.submit.input.RuntimeConfig.Model.ProviderName)
	}
	if engine.submit.input.RuntimeConfig.Tooling.PermissionMode != core.PermissionModePlan {
		t.Fatalf("permission mode = %q", engine.submit.input.RuntimeConfig.Tooling.PermissionMode)
	}
	if engine.submit.input.Metadata["source"] != "http" {
		t.Fatalf("metadata source = %q", engine.submit.input.Metadata["source"])
	}
}

func TestServiceSubmitDoesNotDeriveRuntimeConfigFromMetadata(t *testing.T) {
	engine := &engineStub{runID: "run_plain"}
	svc := NewService(engine)

	resp, err := svc.Submit(context.Background(), SubmitRequest{
		SessionID: "sess_3",
		Input:     "hello",
		Metadata: map[string]string{
			"model_provider": "openai-compatible",
			"model_endpoint": "https://example.invalid/v1",
			"model_name":     "gpt-test",
		},
	})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	if resp.RunID != "run_plain" {
		t.Fatalf("run id = %q, want run_plain", resp.RunID)
	}
	if engine.submit.input.RuntimeConfig != nil {
		t.Fatalf("expected runtime config to remain nil, got %#v", engine.submit.input.RuntimeConfig)
	}
}

func TestServiceControlParsesAction(t *testing.T) {
	engine := &engineStub{}
	svc := NewService(engine)
	if err := svc.Control(context.Background(), ControlRequest{RunID: "run_1", Action: "approve"}); err != nil {
		t.Fatalf("control failed: %v", err)
	}
	if engine.control.req.Action != core.ControlActionApprove {
		t.Fatalf("action = %q, want approve", engine.control.req.Action)
	}

	if err := svc.Control(context.Background(), ControlRequest{
		RunID:  "run_1",
		Action: "answer",
		Answer: &ControlAnswer{
			QuestionID:       "q1",
			SelectedOptionID: "o1",
		},
	}); err != nil {
		t.Fatalf("control answer failed: %v", err)
	}
	if engine.control.req.Answer == nil || engine.control.req.Answer.QuestionID != "q1" {
		t.Fatalf("expected answer payload to be forwarded, got %#v", engine.control.req.Answer)
	}
	if err := svc.Control(context.Background(), ControlRequest{RunID: "run_1", Action: "invalid"}); err == nil {
		t.Fatal("expected invalid action error")
	}
}

func TestServiceSubscribeDelegatesAndEncodesEvent(t *testing.T) {
	sub := &subscriptionStub{events: make(chan core.EventEnvelope, 1)}
	sub.events <- core.EventEnvelope{
		Type:      core.RunEventTypeRunCompleted,
		SessionID: core.SessionID("sess_1"),
		RunID:     core.RunID("run_1"),
		Sequence:  5,
		Timestamp: time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC),
		Payload:   core.RunCompletedPayload{UsageTokens: 17},
	}

	engine := &engineStub{sub: sub}
	svc := NewService(engine)
	frames, err := svc.SubscribeSnapshot(context.Background(), SubscribeRequest{SessionID: "sess_1", Cursor: "3", Limit: 1})
	if err != nil {
		t.Fatalf("subscribe snapshot failed: %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("frames len = %d, want 1", len(frames))
	}
	if frames[0].Type != string(core.RunEventTypeRunCompleted) {
		t.Fatalf("frame type = %q", frames[0].Type)
	}
	if frames[0].Payload["usage_tokens"] != 17 {
		t.Fatalf("payload usage_tokens = %#v", frames[0].Payload["usage_tokens"])
	}
	if engine.subscribe.sessionID != "sess_1" || engine.subscribe.cursor != "3" {
		t.Fatalf("unexpected subscribe delegation: %#v", engine.subscribe)
	}
}

func TestServicePropagatesEngineErrors(t *testing.T) {
	engine := &engineStub{startErr: errors.New("boom")}
	svc := NewService(engine)
	_, err := svc.StartSession(context.Background(), StartSessionRequest{WorkingDir: "/tmp"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceSessionLifecycleRequiresDependency(t *testing.T) {
	svc := NewService(&engineStub{})
	_, err := svc.ResumeSession(context.Background(), ResumeSessionRequest{SessionID: "sess_missing"})
	if !errors.Is(err, ErrSessionLifecycleNotConfigured) {
		t.Fatalf("expected lifecycle missing error, got %v", err)
	}
}

func TestServiceResumeSessionDelegatesLifecycle(t *testing.T) {
	lifecycle := &lifecycleStub{
		state: runtimesession.State{
			SessionID:            core.SessionID("sess_resume"),
			PermissionMode:       core.PermissionModeDefault,
			TemporaryPermissions: []string{"Read(file.md)"},
			HistoryEntries:       7,
			Summary:              "resume summary",
			CreatedAt:            time.Date(2026, 3, 3, 9, 0, 0, 0, time.UTC),
			UpdatedAt:            time.Date(2026, 3, 3, 9, 1, 0, 0, time.UTC),
		},
	}
	svc := NewServiceWithLifecycle(&engineStub{}, lifecycle)
	resp, err := svc.ResumeSession(context.Background(), ResumeSessionRequest{SessionID: "sess_resume"})
	if err != nil {
		t.Fatalf("resume session failed: %v", err)
	}
	if lifecycle.resumeReq.SessionID != core.SessionID("sess_resume") {
		t.Fatalf("unexpected resume request %#v", lifecycle.resumeReq)
	}
	if resp.SessionID != "sess_resume" || resp.HistoryEntries != 7 {
		t.Fatalf("unexpected response %#v", resp)
	}
	if resp.CreatedAt != "2026-03-03T09:00:00Z" || resp.UpdatedAt != "2026-03-03T09:01:00Z" {
		t.Fatalf("unexpected response timestamps %#v", resp)
	}
}

func TestServiceForkRewindClearAndHandoffDelegateLifecycle(t *testing.T) {
	lifecycle := &lifecycleStub{
		state: runtimesession.State{
			SessionID:             core.SessionID("sess_child"),
			ParentSessionID:       core.SessionID("sess_parent"),
			WorkingDir:            "/tmp/child",
			AdditionalDirectories: []string{"/tmp/shared"},
			PermissionMode:        core.PermissionModePlan,
			LastCheckpointID:      core.CheckpointID("cp_2"),
			NextCursor:            5,
			LastClearedReason:     "user_clear",
			CreatedAt:             time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC),
			UpdatedAt:             time.Date(2026, 3, 3, 10, 2, 0, 0, time.UTC),
		},
		snapshot: runtimesession.HandoffSnapshot{
			SessionID:             core.SessionID("sess_child"),
			Target:                runtimesession.HandoffTargetMobile,
			WorkingDir:            "/tmp/child",
			AdditionalDirectories: []string{"/tmp/shared"},
			PermissionMode:        core.PermissionModePlan,
			HistoryEntries:        9,
			Summary:               "ongoing",
			PendingTaskSummary:    "finish adapter",
			LastCheckpointID:      core.CheckpointID("cp_2"),
			NextCursor:            5,
			IssuedAt:              time.Date(2026, 3, 3, 10, 3, 0, 0, time.UTC),
		},
	}

	svc := NewServiceWithLifecycle(&engineStub{}, lifecycle)

	forked, err := svc.ForkSession(context.Background(), ForkSessionRequest{
		SessionID:             "sess_parent",
		WorkingDir:            "/tmp/child",
		AdditionalDirectories: []string{"/tmp/shared", "/tmp/shared"},
	})
	if err != nil {
		t.Fatalf("fork session failed: %v", err)
	}
	if lifecycle.forkReq.SessionID != core.SessionID("sess_parent") {
		t.Fatalf("unexpected fork request %#v", lifecycle.forkReq)
	}
	if len(lifecycle.forkReq.AdditionalDirectories) != 1 {
		t.Fatalf("expected deduplicated additional directories, got %#v", lifecycle.forkReq.AdditionalDirectories)
	}
	if forked.ParentSessionID != "sess_parent" {
		t.Fatalf("unexpected fork response %#v", forked)
	}

	rewound, err := svc.RewindSession(context.Background(), RewindSessionRequest{
		SessionID:            "sess_child",
		CheckpointID:         "cp_2",
		TargetCursor:         5,
		ClearTempPermissions: true,
	})
	if err != nil {
		t.Fatalf("rewind session failed: %v", err)
	}
	if lifecycle.rewindReq.CheckpointID != core.CheckpointID("cp_2") || !lifecycle.rewindReq.ClearTempPerm {
		t.Fatalf("unexpected rewind request %#v", lifecycle.rewindReq)
	}
	if rewound.LastCheckpointID != "cp_2" || rewound.NextCursor != 5 {
		t.Fatalf("unexpected rewind response %#v", rewound)
	}

	cleared, err := svc.ClearSession(context.Background(), ClearSessionRequest{
		SessionID: "sess_child",
		Reason:    " user_clear ",
	})
	if err != nil {
		t.Fatalf("clear session failed: %v", err)
	}
	if lifecycle.clearReq.Reason != "user_clear" {
		t.Fatalf("unexpected clear request %#v", lifecycle.clearReq)
	}
	if cleared.LastClearedReason != "user_clear" {
		t.Fatalf("unexpected clear response %#v", cleared)
	}

	handoff, err := svc.HandoffSession(context.Background(), HandoffSessionRequest{
		SessionID:          "sess_child",
		Target:             "MOBILE",
		PendingTaskSummary: " finish adapter ",
	})
	if err != nil {
		t.Fatalf("handoff session failed: %v", err)
	}
	if lifecycle.handoffReq.Target != runtimesession.HandoffTargetMobile {
		t.Fatalf("unexpected handoff target %#v", lifecycle.handoffReq.Target)
	}
	if handoff.Target != "mobile" || handoff.PendingTaskSummary != "finish adapter" {
		t.Fatalf("unexpected handoff response %#v", handoff)
	}
	if handoff.IssuedAt != "2026-03-03T10:03:00Z" {
		t.Fatalf("unexpected handoff issued at %#v", handoff)
	}
}
