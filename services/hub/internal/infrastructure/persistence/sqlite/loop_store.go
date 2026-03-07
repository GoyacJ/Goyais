package sqlite

import (
	"context"
	"database/sql"
	"time"

	"goyais/services/hub/internal/agent/core"
	"goyais/services/hub/internal/agent/core/statemachine"
	"goyais/services/hub/internal/agent/runtime/loop"
	"goyais/services/hub/internal/domain"
)

type LoopPersistenceStore struct {
	db          *sql.DB
	sessionRepo SessionRepository
	runRepo     RunRepository
}

func NewLoopPersistenceStore(db *sql.DB) *LoopPersistenceStore {
	return &LoopPersistenceStore{
		db:          db,
		sessionRepo: NewSessionRepository(db),
		runRepo:     NewRunRepository(db),
	}
}

func (s *LoopPersistenceStore) SaveSession(ctx context.Context, session loop.PersistedSession) error {
	if s == nil {
		return nil
	}
	return s.sessionRepo.Save(ctx, domain.Session{
		ID:                    domain.SessionID(session.SessionID),
		WorkspaceID:           domain.WorkspaceID(""),
		ProjectID:             "",
		Name:                  string(session.SessionID),
		DefaultMode:           "default",
		ModelConfigID:         "",
		WorkingDir:            session.WorkingDir,
		AdditionalDirectories: append([]string(nil), session.AdditionalDirectories...),
		NextSequence:          session.NextSequence,
		ActiveRunID:           runIDPtr(session.ActiveRunID),
		CreatedAt:             session.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:             session.CreatedAt.UTC().Format(time.RFC3339),
	})
}

func (s *LoopPersistenceStore) SaveRun(ctx context.Context, run loop.PersistedRun) error {
	if s == nil {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return s.runRepo.Save(ctx, domain.Run{
		ID:                    domain.RunID(run.RunID),
		SessionID:             domain.SessionID(run.SessionID),
		WorkspaceID:           domain.WorkspaceID(""),
		State:                 toDomainRunState(run.State),
		InputText:             run.InputText,
		WorkingDir:            run.WorkingDir,
		AdditionalDirectories: append([]string(nil), run.AdditionalDirectories...),
		CreatedAt:             now,
		UpdatedAt:             now,
	})
}

func (s *LoopPersistenceStore) Load(ctx context.Context) (loop.PersistenceSnapshot, error) {
	if s == nil || s.db == nil {
		return loop.PersistenceSnapshot{}, nil
	}

	sessionRows, err := s.db.QueryContext(
		ctx,
		`SELECT id, working_dir, additional_directories_json, next_sequence, active_run_id, created_at
		   FROM domain_sessions
		  ORDER BY created_at ASC, id ASC`,
	)
	if err != nil {
		return loop.PersistenceSnapshot{}, err
	}
	defer sessionRows.Close()

	snapshot := loop.PersistenceSnapshot{
		Sessions: []loop.PersistedSession{},
		Runs:     []loop.PersistedRun{},
	}
	for sessionRows.Next() {
		var (
			sessionID                   core.SessionID
			workingDir                  string
			additionalDirectoriesJSON   string
			nextSequence                int64
			activeRunID                 sql.NullString
			createdAtRaw                string
		)
		if err := sessionRows.Scan(
			&sessionID,
			&workingDir,
			&additionalDirectoriesJSON,
			&nextSequence,
			&activeRunID,
			&createdAtRaw,
		); err != nil {
			return loop.PersistenceSnapshot{}, err
		}
		createdAt, err := time.Parse(time.RFC3339, createdAtRaw)
		if err != nil {
			return loop.PersistenceSnapshot{}, err
		}
		additionalDirectories := []string{}
		if err := decodeStringSlice(additionalDirectoriesJSON, &additionalDirectories); err != nil {
			return loop.PersistenceSnapshot{}, err
		}
		item := loop.PersistedSession{
			SessionID:             sessionID,
			CreatedAt:             createdAt,
			WorkingDir:            workingDir,
			AdditionalDirectories: additionalDirectories,
			NextSequence:          nextSequence,
		}
		if activeRunID.Valid {
			item.ActiveRunID = core.RunID(activeRunID.String)
		}
		snapshot.Sessions = append(snapshot.Sessions, item)
	}
	if err := sessionRows.Err(); err != nil {
		return loop.PersistenceSnapshot{}, err
	}

	runRows, err := s.db.QueryContext(
		ctx,
		`SELECT id, session_id, state, input_text, working_dir, additional_directories_json
		   FROM domain_runs
		  ORDER BY created_at ASC, id ASC`,
	)
	if err != nil {
		return loop.PersistenceSnapshot{}, err
	}
	defer runRows.Close()

	for runRows.Next() {
		var (
			runID                      core.RunID
			sessionID                  core.SessionID
			stateRaw                   string
			inputText                  string
			workingDir                 string
			additionalDirectoriesJSON  string
		)
		if err := runRows.Scan(
			&runID,
			&sessionID,
			&stateRaw,
			&inputText,
			&workingDir,
			&additionalDirectoriesJSON,
		); err != nil {
			return loop.PersistenceSnapshot{}, err
		}
		additionalDirectories := []string{}
		if err := decodeStringSlice(additionalDirectoriesJSON, &additionalDirectories); err != nil {
			return loop.PersistenceSnapshot{}, err
		}
		snapshot.Runs = append(snapshot.Runs, loop.PersistedRun{
			RunID:                 runID,
			SessionID:             sessionID,
			State:                 toStateMachineRunState(domain.RunState(stateRaw)),
			InputText:             inputText,
			WorkingDir:            workingDir,
			AdditionalDirectories: additionalDirectories,
		})
	}
	return snapshot, runRows.Err()
}

func runIDPtr(value core.RunID) *domain.RunID {
	if value == "" {
		return nil
	}
	converted := domain.RunID(value)
	return &converted
}

func toDomainRunState(state statemachine.RunState) domain.RunState {
	switch state {
	case statemachine.RunStateQueued:
		return domain.RunStateQueued
	case statemachine.RunStateRunning:
		return domain.RunStateExecuting
	case statemachine.RunStateCompleted:
		return domain.RunStateCompleted
	case statemachine.RunStateFailed:
		return domain.RunStateFailed
	case statemachine.RunStateCancelled:
		return domain.RunStateCancelled
	case statemachine.RunStateWaitingApproval:
		return domain.RunStateConfirming
	case statemachine.RunStateWaitingUserInput:
		return domain.RunStateAwaiting
	default:
		return domain.RunStateQueued
	}
}

func toStateMachineRunState(state domain.RunState) statemachine.RunState {
	switch state {
	case domain.RunStateQueued:
		return statemachine.RunStateQueued
	case domain.RunStateExecuting:
		return statemachine.RunStateRunning
	case domain.RunStateCompleted:
		return statemachine.RunStateCompleted
	case domain.RunStateFailed:
		return statemachine.RunStateFailed
	case domain.RunStateCancelled:
		return statemachine.RunStateCancelled
	case domain.RunStateConfirming:
		return statemachine.RunStateWaitingApproval
	case domain.RunStateAwaiting:
		return statemachine.RunStateWaitingUserInput
	default:
		return statemachine.RunStateQueued
	}
}
