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
)

type engineStub struct {
	startReq core.StartSessionRequest
	submit   struct {
		sessionID string
		input     core.UserInput
	}
	control struct {
		runID  string
		action core.ControlAction
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

func (s *engineStub) Control(_ context.Context, runID string, action core.ControlAction) error {
	s.control.runID = runID
	s.control.action = action
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

func TestServiceControlParsesAction(t *testing.T) {
	engine := &engineStub{}
	svc := NewService(engine)
	if err := svc.Control(context.Background(), ControlRequest{RunID: "run_1", Action: "approve"}); err != nil {
		t.Fatalf("control failed: %v", err)
	}
	if engine.control.action != core.ControlActionApprove {
		t.Fatalf("action = %q, want approve", engine.control.action)
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
