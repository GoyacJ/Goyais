package domain

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type CreateCheckpointRequest struct {
	SessionID SessionID
	Message   string
}

type CheckpointRepository interface {
	ListSessionCheckpoints(ctx context.Context, sessionID SessionID) ([]Checkpoint, error)
	SaveCheckpoint(ctx context.Context, item StoredCheckpoint) error
	GetCheckpoint(ctx context.Context, sessionID SessionID, checkpointID string) (StoredCheckpoint, bool, error)
}

type CheckpointRuntime interface {
	Capture(ctx context.Context, sessionID SessionID) (CheckpointCapture, error)
	Restore(ctx context.Context, item StoredCheckpoint, strategy CheckpointStrategyKind) (RollbackResult, error)
}

type CheckpointService struct {
	repository CheckpointRepository
	runtime    CheckpointRuntime
	now        func() string
	nextID     func() string
}

type CheckpointServiceOption func(*CheckpointService)

func WithCheckpointNow(now func() string) CheckpointServiceOption {
	return func(service *CheckpointService) {
		if now != nil {
			service.now = now
		}
	}
}

func WithCheckpointID(nextID func() string) CheckpointServiceOption {
	return func(service *CheckpointService) {
		if nextID != nil {
			service.nextID = nextID
		}
	}
}

func NewCheckpointService(repository CheckpointRepository, runtime CheckpointRuntime, options ...CheckpointServiceOption) *CheckpointService {
	service := &CheckpointService{
		repository: repository,
		runtime:    runtime,
		now: func() string {
			return time.Now().UTC().Format(time.RFC3339)
		},
		nextID: func() string {
			return "cp_" + randomHex(8)
		},
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
}

func (s *CheckpointService) ListSessionCheckpoints(ctx context.Context, sessionID SessionID) ([]Checkpoint, error) {
	if s == nil || s.repository == nil {
		return []Checkpoint{}, nil
	}
	return s.repository.ListSessionCheckpoints(ctx, SessionID(strings.TrimSpace(string(sessionID))))
}

func (s *CheckpointService) CreateCheckpoint(ctx context.Context, req CreateCheckpointRequest) (Checkpoint, error) {
	if s == nil || s.repository == nil || s.runtime == nil {
		return Checkpoint{}, nil
	}
	sessionID := SessionID(strings.TrimSpace(string(req.SessionID)))
	message := strings.TrimSpace(req.Message)
	if sessionID == "" {
		return Checkpoint{}, fmt.Errorf("session_id is required")
	}
	if message == "" {
		return Checkpoint{}, fmt.Errorf("message is required")
	}

	parentCheckpointID := ""
	existing, err := s.repository.ListSessionCheckpoints(ctx, sessionID)
	if err != nil {
		return Checkpoint{}, err
	}
	if len(existing) > 0 {
		parentCheckpointID = strings.TrimSpace(existing[0].CheckpointID)
	}

	capture, err := s.runtime.Capture(ctx, sessionID)
	if err != nil {
		return Checkpoint{}, err
	}

	checkpoint := Checkpoint{
		CheckpointID:       strings.TrimSpace(s.nextID()),
		SessionID:          sessionID,
		WorkspaceID:        capture.Session.WorkspaceID,
		ProjectID:          strings.TrimSpace(capture.Session.ProjectID),
		Message:            message,
		ProjectKind:        capture.ProjectKind,
		CreatedAt:          strings.TrimSpace(s.now()),
		GitCommitID:        strings.TrimSpace(capture.GitCommitID),
		EntriesDigest:      strings.TrimSpace(capture.EntriesDigest),
		ParentCheckpointID: parentCheckpointID,
		Session:            cloneCheckpointSessionPtr(&capture.Session),
	}
	if err := s.repository.SaveCheckpoint(ctx, StoredCheckpoint{
		Checkpoint: checkpoint,
		Payload:    strings.TrimSpace(capture.Payload),
	}); err != nil {
		return Checkpoint{}, err
	}
	return checkpoint, nil
}

func (s *CheckpointService) RollbackToCheckpoint(ctx context.Context, sessionID SessionID, checkpointID string) (RollbackResult, error) {
	if s == nil || s.repository == nil || s.runtime == nil {
		return RollbackResult{}, nil
	}
	item, exists, err := s.repository.GetCheckpoint(ctx, SessionID(strings.TrimSpace(string(sessionID))), strings.TrimSpace(checkpointID))
	if err != nil {
		return RollbackResult{}, err
	}
	if !exists {
		return RollbackResult{}, fmt.Errorf("checkpoint not found: %s", strings.TrimSpace(checkpointID))
	}
	result, err := s.runtime.Restore(ctx, item, ResolveCheckpointStrategy(item.Checkpoint.ProjectKind))
	if err != nil {
		return RollbackResult{}, err
	}
	result.Checkpoint = item.Checkpoint
	return result, nil
}

func cloneCheckpointSessionPtr(input *CheckpointSession) *CheckpointSession {
	if input == nil {
		return nil
	}
	copyValue := *input
	copyValue.AdditionalDirectories = append([]string{}, input.AdditionalDirectories...)
	copyValue.TemporaryPermissions = append([]string{}, input.TemporaryPermissions...)
	copyValue.RuleIDs = append([]string{}, input.RuleIDs...)
	copyValue.SkillIDs = append([]string{}, input.SkillIDs...)
	copyValue.MCPIDs = append([]string{}, input.MCPIDs...)
	if input.ActiveExecutionID != nil {
		activeExecutionID := strings.TrimSpace(*input.ActiveExecutionID)
		copyValue.ActiveExecutionID = &activeExecutionID
	}
	return &copyValue
}

func randomHex(bytesLen int) string {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return strings.Repeat("0", bytesLen*2)
	}
	return hex.EncodeToString(buf)
}
