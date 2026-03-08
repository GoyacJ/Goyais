package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	"goyais/services/hub/internal/domain"
)

type checkpointSessionState struct {
	Session        Conversation           `json:"session"`
	Messages       []ConversationMessage  `json:"messages"`
	Snapshots      []ConversationSnapshot `json:"snapshots"`
	Executions     []Execution            `json:"executions"`
	ExecutionOrder []string               `json:"execution_order"`
	ExecutionDiffs map[string][]DiffItem  `json:"execution_diffs"`
}

type checkpointRuntimeMetadata struct {
	RuntimeSessionID      string   `json:"runtime_session_id,omitempty"`
	WorkingDir            string   `json:"working_dir,omitempty"`
	AdditionalDirectories []string `json:"additional_directories,omitempty"`
	TemporaryPermissions  []string `json:"temporary_permissions,omitempty"`
	HistoryEntries        int      `json:"history_entries,omitempty"`
	Summary               string   `json:"summary,omitempty"`
}

type checkpointPayloadEnvelope struct {
	Version      int                       `json:"version"`
	SessionState checkpointSessionState    `json:"session_state"`
	Runtime      checkpointRuntimeMetadata `json:"runtime,omitempty"`
}

const checkpointPayloadVersion = 1

func SessionCheckpointsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := runtimeSessionIDFromPath(r)
		conversation, exists := loadConversationByIDSeed(r.Context(), state, sessionID)
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{
				"session_id": sessionID,
			})
			return
		}

		switch r.Method {
		case http.MethodGet:
			session, authErr := authorizeAction(
				state,
				r,
				conversation.WorkspaceID,
				"session.read",
				authorizationResource{WorkspaceID: conversation.WorkspaceID, ResourceType: "conversation", TargetID: sessionID},
				authorizationContext{OperationType: "read"},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			var (
				items []Checkpoint
				err   error
			)
			if state.checkpointService != nil {
				items, err = state.checkpointService.ListSessionCheckpoints(r.Context(), sessionID)
			} else {
				items, err = listSessionCheckpoints(state, sessionID)
			}
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "CHECKPOINT_LIST_FAILED", "Failed to list checkpoints", map[string]any{
					"session_id": sessionID,
				})
				return
			}
			recordBusinessOperationAudit(r.Context(), state, session, "session.read", "conversation", sessionID, map[string]any{
				"operation": "list_checkpoints",
			})
			writeJSON(w, http.StatusOK, CheckpointListResponse{Items: items})
		case http.MethodPost:
			session, authErr := authorizeAction(
				state,
				r,
				conversation.WorkspaceID,
				"session.write",
				authorizationResource{WorkspaceID: conversation.WorkspaceID, ResourceType: "conversation", TargetID: sessionID},
				authorizationContext{OperationType: "write", ABACRequired: true},
			)
			if authErr != nil {
				authErr.write(w, r)
				return
			}
			input := CheckpointCreateRequest{}
			if err := decodeJSONBody(r, &input); err != nil {
				err.write(w, r)
				return
			}
			message := strings.TrimSpace(input.Message)
			if message == "" {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "message is required", map[string]any{})
				return
			}
			var (
				checkpoint Checkpoint
				err        error
			)
			if state.checkpointService != nil {
				checkpoint, err = state.checkpointService.CreateSessionCheckpoint(r.Context(), sessionID, message)
			} else {
				checkpoint, err = createSessionCheckpoint(state, sessionID, message)
			}
			if err != nil {
				WriteStandardError(w, r, http.StatusInternalServerError, "CHECKPOINT_CREATE_FAILED", "Failed to create checkpoint", map[string]any{
					"session_id": sessionID,
					"error":      err.Error(),
				})
				return
			}
			recordBusinessOperationAudit(r.Context(), state, session, "session.write", "checkpoint", checkpoint.CheckpointID, map[string]any{
				"operation":  "create",
				"session_id": sessionID,
			})
			writeJSON(w, http.StatusCreated, checkpoint)
		default:
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
		}
	}
}

func SessionCheckpointRollbackHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}
		sessionID := runtimeSessionIDFromPath(r)
		checkpointID := strings.TrimSpace(r.PathValue("checkpoint_id"))
		if checkpointID == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "checkpoint_id is required", map[string]any{})
			return
		}
		conversation, exists := loadConversationByIDSeed(r.Context(), state, sessionID)
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{
				"session_id": sessionID,
			})
			return
		}
		session, authErr := authorizeAction(
			state,
			r,
			conversation.WorkspaceID,
			"run.control",
			authorizationResource{WorkspaceID: conversation.WorkspaceID, ResourceType: "conversation", TargetID: sessionID},
			authorizationContext{OperationType: "write", ABACRequired: true},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		var (
			checkpoint      Checkpoint
			restoredSession Conversation
			err             error
		)
		if state.checkpointService != nil {
			checkpoint, restoredSession, err = state.checkpointService.RollbackSessionToCheckpoint(r.Context(), sessionID, checkpointID)
		} else {
			checkpoint, restoredSession, _, err = rollbackSessionToCheckpoint(state, sessionID, checkpointID)
		}
		if err != nil {
			WriteStandardError(w, r, http.StatusNotFound, "CHECKPOINT_NOT_FOUND", "Checkpoint does not exist", map[string]any{
				"session_id":    sessionID,
				"checkpoint_id": checkpointID,
				"error":         err.Error(),
			})
			return
		}
		recordBusinessOperationAudit(r.Context(), state, session, "run.control", "checkpoint", checkpointID, map[string]any{
			"operation":  "rollback",
			"session_id": sessionID,
		})
		writeJSON(w, http.StatusOK, CheckpointRollbackResponse{
			OK:         true,
			Checkpoint: checkpoint,
			Session:    restoredSession,
		})
	}
}

func createSessionCheckpoint(state *AppState, sessionID string, message string) (Checkpoint, error) {
	service := newCheckpointDomainService(state)
	checkpoint, err := service.CreateCheckpoint(context.Background(), domain.CreateCheckpointRequest{
		SessionID: domain.SessionID(strings.TrimSpace(sessionID)),
		Message:   strings.TrimSpace(message),
	})
	if err != nil {
		return Checkpoint{}, err
	}
	return fromDomainCheckpoint(checkpoint), nil
}

func listSessionCheckpoints(state *AppState, sessionID string) ([]Checkpoint, error) {
	service := newCheckpointDomainService(state)
	items, err := service.ListSessionCheckpoints(context.Background(), domain.SessionID(strings.TrimSpace(sessionID)))
	if err != nil {
		return nil, err
	}
	out := make([]Checkpoint, 0, len(items))
	for _, item := range items {
		out = append(out, fromDomainCheckpoint(item))
	}
	return out, nil
}

func rollbackSessionToCheckpoint(state *AppState, sessionID string, checkpointID string) (Checkpoint, Conversation, checkpointRuntimeMetadata, error) {
	service := newCheckpointDomainService(state)
	result, err := service.RollbackToCheckpoint(context.Background(), domain.SessionID(strings.TrimSpace(sessionID)), strings.TrimSpace(checkpointID))
	if err != nil {
		return Checkpoint{}, Conversation{}, checkpointRuntimeMetadata{}, err
	}
	return fromDomainCheckpoint(result.Checkpoint), fromDomainCheckpointSession(result.Session), checkpointRuntimeMetadata{
		RuntimeSessionID:      strings.TrimSpace(result.Runtime.RuntimeSessionID),
		WorkingDir:            strings.TrimSpace(result.Runtime.WorkingDir),
		AdditionalDirectories: append([]string{}, result.Runtime.AdditionalDirectories...),
		TemporaryPermissions:  append([]string{}, result.Runtime.TemporaryPermissions...),
		HistoryEntries:        result.Runtime.HistoryEntries,
		Summary:               strings.TrimSpace(result.Runtime.Summary),
	}, nil
}

func checkpointProjectForSession(state *AppState, sessionID string) (Project, bool, error) {
	conversation, exists := loadConversationByIDSeed(contextWithNoopCancel{}, state, sessionID)
	if !exists {
		return Project{}, false, nil
	}
	return getProjectFromStore(state, conversation.ProjectID)
}

type contextWithNoopCancel struct{}

func (contextWithNoopCancel) Deadline() (time.Time, bool) { return time.Time{}, false }
func (contextWithNoopCancel) Done() <-chan struct{}       { return nil }
func (contextWithNoopCancel) Err() error                  { return nil }
func (contextWithNoopCancel) Value(any) any               { return nil }

func captureCheckpointSessionStateLocked(state *AppState, sessionID string) (checkpointSessionState, bool) {
	conversation, exists := state.conversations[sessionID]
	if !exists {
		return checkpointSessionState{}, false
	}
	executions := listConversationExecutionsLocked(state, sessionID)
	executionDiffs := make(map[string][]DiffItem, len(executions))
	for _, execution := range executions {
		executionDiffs[execution.ID] = append([]DiffItem{}, state.executionDiffs[execution.ID]...)
	}
	return checkpointSessionState{
		Session:        conversation,
		Messages:       cloneMessages(state.conversationMessages[sessionID]),
		Snapshots:      cloneConversationSnapshots(state.conversationSnapshots[sessionID]),
		Executions:     append([]Execution{}, executions...),
		ExecutionOrder: append([]string{}, state.conversationExecutionOrder[sessionID]...),
		ExecutionDiffs: executionDiffs,
	}, true
}

func captureCheckpointRuntimeMetadataLocked(state *AppState, sessionID string, snapshot checkpointSessionState) checkpointRuntimeMetadata {
	metadata := checkpointRuntimeMetadata{
		RuntimeSessionID: strings.TrimSpace(state.conversationSessionIDs[sessionID]),
		HistoryEntries:   len(snapshot.Messages),
	}
	if project, exists := state.projects[snapshot.Session.ProjectID]; exists {
		metadata.WorkingDir = strings.TrimSpace(project.RepoPath)
	}
	return metadata
}

func decodeCheckpointPayload(payload string) (checkpointSessionState, checkpointRuntimeMetadata, error) {
	envelope := checkpointPayloadEnvelope{}
	if err := json.Unmarshal([]byte(payload), &envelope); err == nil && envelope.Version > 0 {
		return envelope.SessionState, normalizeCheckpointRuntimeMetadata(envelope.Runtime, envelope.SessionState), nil
	}

	snapshot := checkpointSessionState{}
	if err := json.Unmarshal([]byte(payload), &snapshot); err != nil {
		return checkpointSessionState{}, checkpointRuntimeMetadata{}, err
	}
	return snapshot, normalizeCheckpointRuntimeMetadata(checkpointRuntimeMetadata{}, snapshot), nil
}

func normalizeCheckpointRuntimeMetadata(input checkpointRuntimeMetadata, snapshot checkpointSessionState) checkpointRuntimeMetadata {
	normalized := checkpointRuntimeMetadata{
		RuntimeSessionID:      strings.TrimSpace(input.RuntimeSessionID),
		WorkingDir:            strings.TrimSpace(input.WorkingDir),
		AdditionalDirectories: uniqueTrimmedStrings(input.AdditionalDirectories),
		TemporaryPermissions:  uniqueTrimmedStrings(input.TemporaryPermissions),
		HistoryEntries:        input.HistoryEntries,
		Summary:               strings.TrimSpace(input.Summary),
	}
	if normalized.HistoryEntries == 0 && len(snapshot.Messages) > 0 {
		normalized.HistoryEntries = len(snapshot.Messages)
	}
	return normalized
}

func restoreCheckpointSessionStateLocked(state *AppState, sessionID string, checkpointID string, snapshot checkpointSessionState, runtimeMetadata checkpointRuntimeMetadata, now string) {
	existingExecutionIDs := append([]string{}, state.conversationExecutionOrder[sessionID]...)
	keepExecutionIDs := make(map[string]struct{}, len(snapshot.ExecutionOrder))
	for _, executionID := range snapshot.ExecutionOrder {
		keepExecutionIDs[strings.TrimSpace(executionID)] = struct{}{}
	}
	for _, executionID := range existingExecutionIDs {
		if _, keep := keepExecutionIDs[strings.TrimSpace(executionID)]; keep {
			continue
		}
		delete(state.executions, executionID)
		delete(state.executionDiffs, executionID)
		delete(state.executionRunIDs, executionID)
	}

	state.conversationMessages[sessionID] = cloneMessages(snapshot.Messages)
	state.conversationSnapshots[sessionID] = cloneConversationSnapshots(snapshot.Snapshots)
	state.conversationExecutionOrder[sessionID] = append([]string{}, snapshot.ExecutionOrder...)
	for _, execution := range snapshot.Executions {
		state.executions[execution.ID] = execution
	}
	for executionID := range state.executionDiffs {
		if _, keep := keepExecutionIDs[strings.TrimSpace(executionID)]; !keep {
			continue
		}
		delete(state.executionDiffs, executionID)
	}
	for executionID, diffItems := range snapshot.ExecutionDiffs {
		state.executionDiffs[executionID] = append([]DiffItem{}, diffItems...)
	}

	session := snapshot.Session
	session.UpdatedAt = now
	state.conversations[sessionID] = session
	if runtimeSessionID := strings.TrimSpace(runtimeMetadata.RuntimeSessionID); runtimeSessionID != "" {
		state.conversationSessionIDs[sessionID] = runtimeSessionID
	} else {
		delete(state.conversationSessionIDs, sessionID)
	}
	rebuildConversationChangeLedgerFromStateLocked(state, sessionID)
	appendExecutionEventLocked(state, ExecutionEvent{
		ExecutionID:    "",
		ConversationID: sessionID,
		TraceID:        GenerateTraceID(),
		QueueIndex:     0,
		Type:           RunEventTypeChangeSetRolledBack,
		Timestamp:      now,
		Payload: map[string]any{
			"checkpoint_id": strings.TrimSpace(checkpointID),
			"source":        "session_checkpoint",
		},
	})
}

func collectCheckpointRollbackExecutionIDsLocked(state *AppState, sessionID string, keptExecutionIDs []string) []string {
	kept := make(map[string]struct{}, len(keptExecutionIDs))
	for _, executionID := range keptExecutionIDs {
		kept[strings.TrimSpace(executionID)] = struct{}{}
	}
	rollbackExecutionIDs := make([]string, 0)
	for executionID, execution := range state.executions {
		if execution.ConversationID != sessionID {
			continue
		}
		if _, exists := kept[executionID]; exists {
			continue
		}
		rollbackExecutionIDs = append(rollbackExecutionIDs, executionID)
	}
	sort.Strings(rollbackExecutionIDs)
	return rollbackExecutionIDs
}

func snapshotChangeEntriesLocked(state *AppState, sessionID string) []ChangeEntry {
	if state == nil {
		return nil
	}
	if ledger := state.conversationChangeLedgers[sessionID]; ledger != nil {
		return append([]ChangeEntry{}, ledger.Entries...)
	}
	return nil
}

func checkpointGitHead(project Project) string {
	if !project.IsGit || !isGitRepositoryPath(project.RepoPath) {
		return ""
	}
	output, err := exec.Command("git", "-C", project.RepoPath, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

type checkpointError string

func (e checkpointError) Error() string { return string(e) }

func errConversationNotFoundForCheckpoint(sessionID string) error {
	return checkpointError("conversation not found for checkpoint: " + strings.TrimSpace(sessionID))
}

func errProjectNotFoundForCheckpoint(sessionID string) error {
	return checkpointError("project not found for session checkpoint: " + strings.TrimSpace(sessionID))
}
