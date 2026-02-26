package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"goyais/services/hub/internal/agentcore/protocol"
	"goyais/services/hub/internal/agentcore/state"
)

type localSession struct {
	ID          string
	ConfigModel string

	events         []protocol.RunEvent
	nextSequence   int64
	subscribers    map[int]chan protocol.RunEvent
	nextSubscriber int
}

type LocalEngine struct {
	mu sync.RWMutex

	sessions     map[string]*localSession
	runToSession map[string]string
}

var simpleArithmeticPattern = regexp.MustCompile(`(-?\d+)\s*([+\-x*/])\s*(-?\d+)`)

func NewLocalEngine() *LocalEngine {
	return &LocalEngine{
		sessions:     map[string]*localSession{},
		runToSession: map[string]string{},
	}
}

func (e *LocalEngine) StartSession(_ context.Context, req StartSessionRequest) (SessionHandle, error) {
	if e == nil {
		return SessionHandle{}, errors.New("local engine is nil")
	}
	if err := req.Validate(); err != nil {
		return SessionHandle{}, err
	}

	sessionID := "sess_" + randomID()
	session := &localSession{
		ID:             sessionID,
		ConfigModel:    strings.TrimSpace(req.Config.DefaultModel),
		events:         make([]protocol.RunEvent, 0, 8),
		nextSequence:   0,
		subscribers:    map[int]chan protocol.RunEvent{},
		nextSubscriber: 1,
	}

	e.mu.Lock()
	e.sessions[sessionID] = session
	e.mu.Unlock()

	return SessionHandle{SessionID: sessionID}, nil
}

func (e *LocalEngine) Submit(_ context.Context, sessionID string, input UserInput) (string, error) {
	if e == nil {
		return "", errors.New("local engine is nil")
	}
	if err := input.Validate(); err != nil {
		return "", err
	}

	normalizedSessionID := strings.TrimSpace(sessionID)
	if normalizedSessionID == "" {
		return "", errors.New("session_id is required")
	}

	e.mu.RLock()
	session, exists := e.sessions[normalizedSessionID]
	e.mu.RUnlock()
	if !exists {
		return "", fmt.Errorf("session %q does not exist", normalizedSessionID)
	}

	runID := "run_" + randomID()
	e.mu.Lock()
	e.runToSession[runID] = normalizedSessionID
	e.mu.Unlock()

	trimmed := strings.TrimSpace(input.Text)
	response := localDeterministicResponse(trimmed)
	e.emitEvent(normalizedSessionID, runID, protocol.RunEventTypeRunQueued, map[string]any{
		"model": session.ConfigModel,
	})
	e.emitEvent(normalizedSessionID, runID, protocol.RunEventTypeRunStarted, map[string]any{
		"source": "local_engine",
	})
	e.emitEvent(normalizedSessionID, runID, protocol.RunEventTypeRunOutputDelta, map[string]any{
		"delta":   response,
		"output":  response,
		"content": response,
	})
	e.emitEvent(normalizedSessionID, runID, protocol.RunEventTypeRunCompleted, map[string]any{
		"source": "local_engine",
	})

	return runID, nil
}

func (e *LocalEngine) Control(_ context.Context, runID string, action state.ControlAction) error {
	if e == nil {
		return errors.New("local engine is nil")
	}

	normalizedRunID := strings.TrimSpace(runID)
	if normalizedRunID == "" {
		return errors.New("run_id is required")
	}

	e.mu.RLock()
	sessionID, exists := e.runToSession[normalizedRunID]
	e.mu.RUnlock()
	if !exists {
		return fmt.Errorf("run %q does not exist", normalizedRunID)
	}

	switch action {
	case state.ControlActionStop, state.ControlActionDeny:
		e.emitEvent(sessionID, normalizedRunID, protocol.RunEventTypeRunCancelled, map[string]any{
			"action": string(action),
		})
	case state.ControlActionApprove, state.ControlActionResume:
		e.emitEvent(sessionID, normalizedRunID, protocol.RunEventTypeRunStarted, map[string]any{
			"action": string(action),
		})
	default:
		return fmt.Errorf("unsupported control action %q", action)
	}

	return nil
}

func (e *LocalEngine) Subscribe(_ context.Context, sessionID string, cursor string) (<-chan protocol.RunEvent, error) {
	if e == nil {
		return nil, errors.New("local engine is nil")
	}

	normalizedSessionID := strings.TrimSpace(sessionID)
	if normalizedSessionID == "" {
		return nil, errors.New("session_id is required")
	}

	minSequence := int64(-1)
	if trimmed := strings.TrimSpace(cursor); trimmed != "" {
		parsed, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor %q", cursor)
		}
		minSequence = parsed
	}

	e.mu.Lock()
	session, exists := e.sessions[normalizedSessionID]
	if !exists {
		e.mu.Unlock()
		return nil, fmt.Errorf("session %q does not exist", normalizedSessionID)
	}

	ch := make(chan protocol.RunEvent, 64)
	subscriberID := session.nextSubscriber
	session.nextSubscriber++
	session.subscribers[subscriberID] = ch

	replay := make([]protocol.RunEvent, 0, len(session.events))
	for _, event := range session.events {
		if event.Sequence > minSequence {
			replay = append(replay, event)
		}
	}
	e.mu.Unlock()

	go func() {
		for _, event := range replay {
			ch <- event
		}
	}()

	return ch, nil
}

func (e *LocalEngine) emitEvent(sessionID string, runID string, eventType protocol.RunEventType, payload map[string]any) {
	e.mu.Lock()
	session, exists := e.sessions[sessionID]
	if !exists {
		e.mu.Unlock()
		return
	}

	sequence := session.nextSequence
	session.nextSequence++

	event := protocol.RunEvent{
		Type:      eventType,
		SessionID: sessionID,
		RunID:     runID,
		Sequence:  sequence,
		Timestamp: time.Now().UTC(),
		Payload:   cloneAnyMap(payload),
	}
	session.events = append(session.events, event)

	subscribers := make([]chan protocol.RunEvent, 0, len(session.subscribers))
	for _, ch := range session.subscribers {
		subscribers = append(subscribers, ch)
	}
	e.mu.Unlock()

	for _, ch := range subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func randomID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

// localDeterministicResponse keeps local/test runs reproducible without echoing
// user prompts. This mirrors a minimal "answer-first" runtime behavior for regression coverage.
func localDeterministicResponse(prompt string) string {
	if result, ok := solveSimpleArithmetic(prompt); ok {
		return result
	}
	return "I can help with that. Please provide a concrete question."
}

func solveSimpleArithmetic(prompt string) (string, bool) {
	match := simpleArithmeticPattern.FindStringSubmatch(prompt)
	if len(match) != 4 {
		return "", false
	}

	left, err := strconv.Atoi(strings.TrimSpace(match[1]))
	if err != nil {
		return "", false
	}
	right, err := strconv.Atoi(strings.TrimSpace(match[3]))
	if err != nil {
		return "", false
	}

	switch strings.TrimSpace(match[2]) {
	case "+":
		return strconv.Itoa(left + right), true
	case "-":
		return strconv.Itoa(left - right), true
	case "*", "x":
		return strconv.Itoa(left * right), true
	case "/":
		if right == 0 {
			return "", false
		}
		return strconv.Itoa(left / right), true
	default:
		return "", false
	}
}
