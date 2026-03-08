package httpapi

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"
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
	project, projectExists, projectErr := checkpointProjectForSession(state, sessionID)
	if projectErr != nil {
		return Checkpoint{}, projectErr
	}
	if !projectExists {
		return Checkpoint{}, errProjectNotFoundForCheckpoint(sessionID)
	}

	var (
		checkpoint Checkpoint
		payload    string
	)
	parentCheckpointID := ""
	existingCheckpoints, err := listSessionCheckpoints(state, sessionID)
	if err != nil {
		return Checkpoint{}, err
	}
	if len(existingCheckpoints) > 0 {
		parentCheckpointID = strings.TrimSpace(existingCheckpoints[0].CheckpointID)
	}
	state.mu.Lock()
	defer state.mu.Unlock()

	snapshot, exists := captureCheckpointSessionStateLocked(state, sessionID)
	if !exists {
		return Checkpoint{}, errConversationNotFoundForCheckpoint(sessionID)
	}
	encoded, err := json.Marshal(checkpointPayloadEnvelope{
		Version:      checkpointPayloadVersion,
		SessionState: snapshot,
		Runtime:      captureCheckpointRuntimeMetadataLocked(state, sessionID, snapshot),
	})
	if err != nil {
		return Checkpoint{}, err
	}
	payload = string(encoded)

	createdAt := time.Now().UTC().Format(time.RFC3339)
	checkpoint = Checkpoint{
		CheckpointSummary: CheckpointSummary{
			CheckpointID:  "cp_" + randomHex(8),
			Message:       strings.TrimSpace(message),
			ProjectKind:   checkpointProjectKind(project.IsGit),
			CreatedAt:     createdAt,
			GitCommitID:   checkpointGitHead(project),
			EntriesDigest: digestChangeEntries(snapshotChangeEntriesLocked(state, sessionID)),
		},
		SessionID:          sessionID,
		ParentCheckpointID: parentCheckpointID,
		Session:            cloneConversationPtr(&snapshot.Session),
	}
	appendCheckpointLocked(state, checkpoint, payload)
	return checkpoint, persistCheckpointLocked(state, checkpoint, payload)
}

func listSessionCheckpoints(state *AppState, sessionID string) ([]Checkpoint, error) {
	if state == nil {
		return []Checkpoint{}, nil
	}
	if state.authz != nil {
		rows, err := state.authz.listSessionCheckpoints(sessionID)
		if err != nil {
			return nil, err
		}
		items := make([]Checkpoint, 0, len(rows))
		for _, row := range rows {
			checkpoint := row.Checkpoint
			if checkpoint.Session == nil {
				checkpoint.Session = &Conversation{ID: sessionID}
			}
			items = append(items, checkpoint)
		}
		return items, nil
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	items := append([]Checkpoint{}, state.conversationCheckpoints[sessionID]...)
	return items, nil
}

func rollbackSessionToCheckpoint(state *AppState, sessionID string, checkpointID string) (Checkpoint, Conversation, checkpointRuntimeMetadata, error) {
	checkpoint, payload, exists, err := loadCheckpointPayload(state, sessionID, checkpointID)
	if err != nil {
		return Checkpoint{}, Conversation{}, checkpointRuntimeMetadata{}, err
	}
	if !exists {
		return Checkpoint{}, Conversation{}, checkpointRuntimeMetadata{}, errCheckpointNotFound(checkpointID)
	}
	snapshot, runtimeMetadata, err := decodeCheckpointPayload(payload)
	if err != nil {
		return Checkpoint{}, Conversation{}, checkpointRuntimeMetadata{}, err
	}

	project, projectExists, projectErr := checkpointProjectForSession(state, sessionID)
	if projectErr != nil {
		return Checkpoint{}, Conversation{}, checkpointRuntimeMetadata{}, projectErr
	}
	if !projectExists {
		return Checkpoint{}, Conversation{}, checkpointRuntimeMetadata{}, errProjectNotFoundForCheckpoint(sessionID)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	state.mu.Lock()
	rollbackExecutionIDs := collectCheckpointRollbackExecutionIDsLocked(state, sessionID, snapshot.ExecutionOrder)
	rollbackDiffItems := make([]DiffItem, 0)
	rollbackExecutionIDSet := make(map[string]struct{}, len(rollbackExecutionIDs))
	for _, executionID := range rollbackExecutionIDs {
		rollbackExecutionIDSet[executionID] = struct{}{}
		rollbackDiffItems = mergeDiffItems(rollbackDiffItems, state.executionDiffs[executionID])
	}
	rollbackEntries := changeEntriesFromDiffItems(rollbackDiffItems)
	if len(rollbackEntries) == 0 {
		if ledger := state.conversationChangeLedgers[sessionID]; ledger != nil {
			for _, entry := range ledger.Entries {
				if _, exists := rollbackExecutionIDSet[strings.TrimSpace(entry.ExecutionID)]; exists {
					rollbackEntries = append(rollbackEntries, entry)
				}
			}
		}
	}
	state.mu.Unlock()

	projectSupportsGitRestore := project.IsGit && isGitRepositoryPath(project.RepoPath)
	if projectSupportsGitRestore && strings.TrimSpace(project.RepoPath) != "" && len(rollbackDiffItems) > 0 {
		if err := restoreGitWorkingTreePaths(project.RepoPath, rollbackDiffItems); err != nil {
			return Checkpoint{}, Conversation{}, checkpointRuntimeMetadata{}, err
		}
	}
	if !projectSupportsGitRestore && strings.TrimSpace(project.RepoPath) != "" && len(rollbackEntries) > 0 {
		if err := restoreNonGitWorkingTreePaths(project.RepoPath, rollbackEntries); err != nil {
			return Checkpoint{}, Conversation{}, checkpointRuntimeMetadata{}, err
		}
	}

	state.mu.Lock()
	restoreCheckpointSessionStateLocked(state, sessionID, checkpointID, snapshot, runtimeMetadata, now)
	state.mu.Unlock()
	syncExecutionDomainBestEffort(state)

	return checkpoint, snapshot.Session, runtimeMetadata, nil
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

func appendCheckpointLocked(state *AppState, checkpoint Checkpoint, payload string) {
	state.conversationCheckpoints[checkpoint.SessionID] = append([]Checkpoint{checkpoint}, state.conversationCheckpoints[checkpoint.SessionID]...)
	state.checkpointSessionPayloads[checkpoint.CheckpointID] = payload
}

func persistCheckpointLocked(state *AppState, checkpoint Checkpoint, payload string) error {
	if state.authz == nil {
		return nil
	}
	return state.authz.insertSessionCheckpoint(storedSessionCheckpoint{
		Checkpoint:  checkpoint,
		SessionJSON: payload,
	})
}

func loadCheckpointPayload(state *AppState, sessionID string, checkpointID string) (Checkpoint, string, bool, error) {
	if state == nil {
		return Checkpoint{}, "", false, nil
	}
	if state.authz != nil {
		row, exists, err := state.authz.getSessionCheckpoint(sessionID, checkpointID)
		if err != nil || !exists {
			return Checkpoint{}, "", exists, err
		}
		return row.Checkpoint, row.SessionJSON, true, nil
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	payload, ok := state.checkpointSessionPayloads[checkpointID]
	if !ok {
		return Checkpoint{}, "", false, nil
	}
	for _, checkpoint := range state.conversationCheckpoints[sessionID] {
		if checkpoint.CheckpointID == checkpointID {
			return checkpoint, payload, true, nil
		}
	}
	return Checkpoint{}, "", false, nil
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

func checkpointProjectKind(isGit bool) string {
	if isGit {
		return "git"
	}
	return "non_git"
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

func cloneConversationPtr(input *Conversation) *Conversation {
	if input == nil {
		return nil
	}
	copyValue := *input
	if input.ActiveExecutionID != nil {
		active := strings.TrimSpace(*input.ActiveExecutionID)
		copyValue.ActiveExecutionID = &active
	}
	copyValue.RuleIDs = append([]string{}, input.RuleIDs...)
	copyValue.SkillIDs = append([]string{}, input.SkillIDs...)
	copyValue.MCPIDs = append([]string{}, input.MCPIDs...)
	return &copyValue
}

type checkpointError string

func (e checkpointError) Error() string { return string(e) }

func errConversationNotFoundForCheckpoint(sessionID string) error {
	return checkpointError("conversation not found for checkpoint: " + strings.TrimSpace(sessionID))
}

func errProjectNotFoundForCheckpoint(sessionID string) error {
	return checkpointError("project not found for session checkpoint: " + strings.TrimSpace(sessionID))
}

func errCheckpointNotFound(checkpointID string) error {
	return checkpointError("checkpoint not found: " + strings.TrimSpace(checkpointID))
}
