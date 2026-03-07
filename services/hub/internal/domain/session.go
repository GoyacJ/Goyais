package domain

import "fmt"

type SessionID string

type Session struct {
	ID                    SessionID
	WorkspaceID           WorkspaceID
	ProjectID             string
	Name                  string
	DefaultMode           string
	ModelConfigID         string
	WorkingDir            string
	AdditionalDirectories []string
	RuleIDs               []string
	SkillIDs              []string
	MCPIDs                []string
	NextSequence          int64
	ActiveRunID           *RunID
	CreatedAt             string
	UpdatedAt             string
}

func (s *Session) QueueRun(runID RunID) error {
	if s == nil {
		return fmt.Errorf("queue run: session is nil")
	}
	if s.ActiveRunID != nil && *s.ActiveRunID != "" {
		return fmt.Errorf("queue run: active run %s already exists", *s.ActiveRunID)
	}
	normalized := runID
	if normalized == "" {
		return fmt.Errorf("queue run: run id is required")
	}
	s.ActiveRunID = &normalized
	return nil
}

func (s *Session) AdvanceSequence() int64 {
	if s == nil {
		return 0
	}
	current := s.NextSequence
	s.NextSequence++
	return current
}
