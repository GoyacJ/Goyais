package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ConversationChangeLedger struct {
	ConversationID          string
	ProjectKind             string
	PendingChangeSetID      string
	Entries                 []ChangeEntry
	LastCommittedCheckpoint *CheckpointSummary
}

type ProjectChangeDriver interface {
	Kind() string
	Commit(project Project, entries []ChangeEntry, message string) (CheckpointSummary, error)
	Discard(project Project, entries []ChangeEntry) error
	Export(project Project, entries []ChangeEntry) (ExecutionFilesExportResponse, error)
}

type gitProjectChangeDriver struct{}
type nonGitProjectChangeDriver struct{}

func (gitProjectChangeDriver) Kind() string {
	return "git"
}

func (gitProjectChangeDriver) Commit(project Project, entries []ChangeEntry, message string) (CheckpointSummary, error) {
	repoPath := strings.TrimSpace(project.RepoPath)
	if repoPath == "" {
		return CheckpointSummary{}, errors.New("project repo path is empty")
	}
	paths := uniqueEntryPaths(entries)
	if len(paths) == 0 {
		return CheckpointSummary{}, errors.New("no pending changes to commit")
	}
	addArgs := []string{"-C", repoPath, "add", "--"}
	addArgs = append(addArgs, paths...)
	if output, err := exec.Command("git", addArgs...).CombinedOutput(); err != nil {
		return CheckpointSummary{}, fmt.Errorf("git add failed: %s", strings.TrimSpace(string(output)))
	}
	commitArgs := []string{"-C", repoPath, "commit", "-m", message, "--"}
	commitArgs = append(commitArgs, paths...)
	if output, err := exec.Command("git", commitArgs...).CombinedOutput(); err != nil {
		return CheckpointSummary{}, fmt.Errorf("git commit failed: %s", strings.TrimSpace(string(output)))
	}
	headOutput, err := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		return CheckpointSummary{}, fmt.Errorf("resolve git commit id failed: %w", err)
	}
	commitID := strings.TrimSpace(string(headOutput))
	now := time.Now().UTC().Format(time.RFC3339)
	return CheckpointSummary{
		CheckpointID:  "cp_git_" + randomHex(6),
		Message:       message,
		CreatedAt:     now,
		GitCommitID:   commitID,
		EntriesDigest: digestChangeEntries(entries),
	}, nil
}

func (gitProjectChangeDriver) Discard(project Project, entries []ChangeEntry) error {
	repoPath := strings.TrimSpace(project.RepoPath)
	if repoPath == "" {
		return errors.New("project repo path is empty")
	}
	diffItems := make([]DiffItem, 0, len(entries))
	for _, entry := range entries {
		diffItems = append(diffItems, DiffItem{
			Path:       entry.Path,
			ChangeType: entry.ChangeType,
		})
	}
	return restoreGitWorkingTreePaths(repoPath, diffItems)
}

func (gitProjectChangeDriver) Export(project Project, entries []ChangeEntry) (ExecutionFilesExportResponse, error) {
	repoPath := strings.TrimSpace(project.RepoPath)
	if repoPath == "" {
		return ExecutionFilesExportResponse{}, errors.New("project repo path is empty")
	}
	archiveBase64, err := renderChangeSetFilesArchiveBase64(repoPath, entries)
	if err != nil {
		return ExecutionFilesExportResponse{}, err
	}
	return ExecutionFilesExportResponse{
		FileName:      fmt.Sprintf("changeset-%s.zip", randomHex(4)),
		ArchiveBase64: archiveBase64,
	}, nil
}

func (nonGitProjectChangeDriver) Kind() string {
	return "non_git"
}

func (nonGitProjectChangeDriver) Commit(_ Project, entries []ChangeEntry, message string) (CheckpointSummary, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	return CheckpointSummary{
		CheckpointID:  "cp_non_git_" + randomHex(6),
		Message:       message,
		CreatedAt:     now,
		EntriesDigest: digestChangeEntries(entries),
	}, nil
}

func (nonGitProjectChangeDriver) Discard(project Project, entries []ChangeEntry) error {
	repoPath := strings.TrimSpace(project.RepoPath)
	if repoPath == "" {
		return errors.New("project repo path is empty")
	}
	return restoreNonGitWorkingTreePaths(repoPath, entries)
}

func (nonGitProjectChangeDriver) Export(project Project, entries []ChangeEntry) (ExecutionFilesExportResponse, error) {
	repoPath := strings.TrimSpace(project.RepoPath)
	if repoPath == "" {
		return ExecutionFilesExportResponse{}, errors.New("project repo path is empty")
	}
	archiveBase64, err := renderChangeSetFilesArchiveBase64(repoPath, entries)
	if err != nil {
		return ExecutionFilesExportResponse{}, err
	}
	return ExecutionFilesExportResponse{
		FileName:      fmt.Sprintf("changeset-%s.zip", randomHex(4)),
		ArchiveBase64: archiveBase64,
	}, nil
}

func ConversationChangeSetHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		conversationID := strings.TrimSpace(r.PathValue("conversation_id"))
		conversationSeed, exists := loadChangeSetConversationSeed(r.Context(), state, conversationID)
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
			return
		}
		projectSeed, projectExists, projectErr := getProjectFromStore(state, conversationSeed.ProjectID)
		if projectErr != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_READ_FAILED", "Failed to read project", map[string]any{
				"project_id": conversationSeed.ProjectID,
			})
			return
		}
		if !projectExists {
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{"project_id": conversationSeed.ProjectID})
			return
		}
		_, authErr := authorizeAction(
			state,
			r,
			conversationSeed.WorkspaceID,
			"conversation.read",
			authorizationResource{WorkspaceID: conversationSeed.WorkspaceID},
			authorizationContext{OperationType: "read"},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		state.mu.Lock()
		conversation, _, projectReady := ensureChangeSetContextSeedsLocked(state, conversationSeed, projectSeed)
		if !projectReady {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{"project_id": conversation.ProjectID})
			return
		}
		changeSet, err := buildConversationChangeSetLocked(state, conversation.ID)
		state.mu.Unlock()
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "CHANGESET_BUILD_FAILED", "Failed to build conversation changeset", map[string]any{
				"conversation_id": conversationID,
				"error":           err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, changeSet)
	}
}

func ConversationChangeSetCommitHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		conversationID := strings.TrimSpace(r.PathValue("conversation_id"))
		input := ChangeSetCommitRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		conversationSeed, exists := loadChangeSetConversationSeed(r.Context(), state, conversationID)
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
			return
		}
		projectSeed, projectExists, projectErr := getProjectFromStore(state, conversationSeed.ProjectID)
		if projectErr != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_READ_FAILED", "Failed to read project", map[string]any{
				"project_id": conversationSeed.ProjectID,
			})
			return
		}
		if !projectExists {
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{"project_id": conversationSeed.ProjectID})
			return
		}
		_, authErr := authorizeAction(
			state,
			r,
			conversationSeed.WorkspaceID,
			"execution.control",
			authorizationResource{WorkspaceID: conversationSeed.WorkspaceID},
			authorizationContext{OperationType: "write", ABACRequired: true},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		state.mu.Lock()
		conversation, project, projectReady := ensureChangeSetContextSeedsLocked(state, conversationSeed, projectSeed)
		if !projectReady {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{"project_id": conversation.ProjectID})
			return
		}
		ledger := ensureConversationChangeLedgerLocked(state, conversationID)
		if hasMutableExecutionsLocked(state, conversationID) {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusConflict, "CHANGESET_BUSY", "Changeset mutation is disabled while executions are running", map[string]any{"conversation_id": conversationID})
			return
		}
		if strings.TrimSpace(input.ExpectedChangeSetID) == "" || strings.TrimSpace(input.ExpectedChangeSetID) != strings.TrimSpace(ledger.PendingChangeSetID) {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusConflict, "CHANGESET_CONFLICT", "Expected change_set_id does not match current pending set", map[string]any{"conversation_id": conversationID})
			return
		}
		entries := cloneChangeEntries(ledger.Entries)
		if len(entries) == 0 {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusConflict, "CHANGESET_EMPTY", "No pending changes to commit", map[string]any{"conversation_id": conversationID})
			return
		}
		message := strings.TrimSpace(input.Message)
		if message == "" {
			message = suggestCommitMessage(entries, resolveProjectKind(project))
		}
		driver := changeDriverForProject(project)
		state.mu.Unlock()

		checkpoint, err := driver.Commit(project, entries, message)
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "CHANGESET_COMMIT_FAILED", "Failed to commit changeset", map[string]any{
				"conversation_id": conversationID,
				"error":           err.Error(),
			})
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		state.mu.Lock()
		ledger = ensureConversationChangeLedgerLocked(state, conversationID)
		ledger.Entries = []ChangeEntry{}
		ledger.LastCommittedCheckpoint = &checkpoint
		bumpPendingChangeSetIDLocked(ledger)
		if projectFresh, ok := state.projects[project.ID]; ok {
			projectFresh.CurrentRevision += 1
			projectFresh.UpdatedAt = now
			state.projects[projectFresh.ID] = projectFresh
			if conversationFresh, exists := state.conversations[conversationID]; exists {
				conversationFresh.BaseRevision = projectFresh.CurrentRevision
				conversationFresh.UpdatedAt = now
				state.conversations[conversationID] = conversationFresh
			}
		}
		appendExecutionEventLocked(state, ExecutionEvent{
			ExecutionID:    "",
			ConversationID: conversationID,
			TraceID:        TraceIDFromContext(r.Context()),
			QueueIndex:     0,
			Type:           RunEventTypeChangeSetCommitted,
			Timestamp:      now,
			Payload: map[string]any{
				"change_set_id": ledger.PendingChangeSetID,
				"checkpoint": map[string]any{
					"checkpoint_id":  checkpoint.CheckpointID,
					"message":        checkpoint.Message,
					"created_at":     checkpoint.CreatedAt,
					"git_commit_id":  checkpoint.GitCommitID,
					"entries_digest": checkpoint.EntriesDigest,
				},
			},
		})
		state.mu.Unlock()
		syncExecutionDomainBestEffort(state)

		writeJSON(w, http.StatusOK, ChangeSetCommitResponse{
			OK:         true,
			Checkpoint: checkpoint,
		})
	}
}

func ConversationChangeSetDiscardHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		conversationID := strings.TrimSpace(r.PathValue("conversation_id"))
		input := ChangeSetDiscardRequest{}
		if err := decodeJSONBody(r, &input); err != nil {
			err.write(w, r)
			return
		}
		conversationSeed, exists := loadChangeSetConversationSeed(r.Context(), state, conversationID)
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
			return
		}
		projectSeed, projectExists, projectErr := getProjectFromStore(state, conversationSeed.ProjectID)
		if projectErr != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_READ_FAILED", "Failed to read project", map[string]any{
				"project_id": conversationSeed.ProjectID,
			})
			return
		}
		if !projectExists {
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{"project_id": conversationSeed.ProjectID})
			return
		}
		state.mu.Lock()
		conversation, project, projectReady := ensureChangeSetContextSeedsLocked(state, conversationSeed, projectSeed)
		if !projectReady {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{"project_id": conversation.ProjectID})
			return
		}
		ledger := ensureConversationChangeLedgerLocked(state, conversationID)
		if hasMutableExecutionsLocked(state, conversationID) {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusConflict, "CHANGESET_BUSY", "Changeset mutation is disabled while executions are running", map[string]any{"conversation_id": conversationID})
			return
		}
		if strings.TrimSpace(input.ExpectedChangeSetID) == "" || strings.TrimSpace(input.ExpectedChangeSetID) != strings.TrimSpace(ledger.PendingChangeSetID) {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusConflict, "CHANGESET_CONFLICT", "Expected change_set_id does not match current pending set", map[string]any{"conversation_id": conversationID})
			return
		}
		entries := cloneChangeEntries(ledger.Entries)
		state.mu.Unlock()
		if len(entries) == 0 {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}

		driver := changeDriverForProject(project)
		if err := driver.Discard(project, entries); err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "CHANGESET_DISCARD_FAILED", "Failed to discard changeset", map[string]any{
				"conversation_id": conversationID,
				"error":           err.Error(),
			})
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		state.mu.Lock()
		ledger = ensureConversationChangeLedgerLocked(state, conversationID)
		ledger.Entries = []ChangeEntry{}
		bumpPendingChangeSetIDLocked(ledger)
		appendExecutionEventLocked(state, ExecutionEvent{
			ExecutionID:    "",
			ConversationID: conversationID,
			TraceID:        TraceIDFromContext(r.Context()),
			QueueIndex:     0,
			Type:           RunEventTypeChangeSetDiscarded,
			Timestamp:      now,
			Payload: map[string]any{
				"change_set_id": ledger.PendingChangeSetID,
			},
		})
		state.mu.Unlock()
		syncExecutionDomainBestEffort(state)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func ConversationChangeSetExportHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		conversationID := strings.TrimSpace(r.PathValue("conversation_id"))
		conversationSeed, exists := loadChangeSetConversationSeed(r.Context(), state, conversationID)
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation does not exist", map[string]any{"conversation_id": conversationID})
			return
		}
		projectSeed, projectExists, projectErr := getProjectFromStore(state, conversationSeed.ProjectID)
		if projectErr != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_READ_FAILED", "Failed to read project", map[string]any{
				"project_id": conversationSeed.ProjectID,
			})
			return
		}
		if !projectExists {
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{"project_id": conversationSeed.ProjectID})
			return
		}
		state.mu.Lock()
		conversation, project, projectReady := ensureChangeSetContextSeedsLocked(state, conversationSeed, projectSeed)
		if !projectReady {
			state.mu.Unlock()
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{"project_id": conversation.ProjectID})
			return
		}
		entries := cloneChangeEntries(ensureConversationChangeLedgerLocked(state, conversationID).Entries)
		state.mu.Unlock()
		_, authErr := authorizeAction(
			state,
			r,
			conversation.WorkspaceID,
			"conversation.read",
			authorizationResource{WorkspaceID: conversation.WorkspaceID},
			authorizationContext{OperationType: "read"},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}
		driver := changeDriverForProject(project)
		payload, err := driver.Export(project, entries)
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "CHANGESET_EXPORT_FAILED", "Failed to export changeset files", map[string]any{
				"conversation_id": conversationID,
				"error":           err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func changeDriverForProject(project Project) ProjectChangeDriver {
	if resolveProjectKind(project) == "git" {
		return gitProjectChangeDriver{}
	}
	return nonGitProjectChangeDriver{}
}

func applyExecutionEventToChangeLedgerLocked(state *AppState, event ExecutionEvent) {
	if state == nil {
		return
	}
	conversationID := strings.TrimSpace(event.ConversationID)
	if conversationID == "" {
		return
	}
	ledger := ensureConversationChangeLedgerLocked(state, conversationID)
	switch event.Type {
	case RunEventTypeDiffGenerated:
		diffItems := parseDiffItemsFromPayload(event.Payload)
		if len(diffItems) == 0 {
			return
		}
		execution, exists := loadChangeSetExecutionSeedLocked(state, strings.TrimSpace(event.ExecutionID))
		messageID := ""
		if exists {
			messageID = execution.MessageID
		}
		for _, item := range diffItems {
			upsertChangeEntryInLedger(ledger, event, messageID, item)
		}
		bumpPendingChangeSetIDLocked(ledger)
	case RunEventTypeChangeSetCommitted:
		ledger.Entries = []ChangeEntry{}
		if checkpoint, ok := checkpointFromPayload(event.Payload); ok {
			ledger.LastCommittedCheckpoint = &checkpoint
		}
		if changeSetID := strings.TrimSpace(asStringValue(event.Payload["change_set_id"])); changeSetID != "" {
			ledger.PendingChangeSetID = changeSetID
		} else {
			bumpPendingChangeSetIDLocked(ledger)
		}
	case RunEventTypeChangeSetDiscarded:
		ledger.Entries = []ChangeEntry{}
		if changeSetID := strings.TrimSpace(asStringValue(event.Payload["change_set_id"])); changeSetID != "" {
			ledger.PendingChangeSetID = changeSetID
		} else {
			bumpPendingChangeSetIDLocked(ledger)
		}
	case RunEventTypeChangeSetRolledBack:
		rebuildConversationChangeLedgerFromStateLocked(state, conversationID)
	default:
		return
	}
}

func ensureConversationChangeLedgerLocked(state *AppState, conversationID string) *ConversationChangeLedger {
	normalizedConversationID := strings.TrimSpace(conversationID)
	if normalizedConversationID == "" {
		return &ConversationChangeLedger{}
	}
	if existing, exists := state.conversationChangeLedgers[normalizedConversationID]; exists && existing != nil {
		if strings.TrimSpace(existing.PendingChangeSetID) == "" {
			existing.PendingChangeSetID = "cs_" + randomHex(6)
		}
		if strings.TrimSpace(existing.ProjectKind) == "" {
			existing.ProjectKind = resolveConversationProjectKindLocked(state, normalizedConversationID)
		}
		return existing
	}
	ledger := &ConversationChangeLedger{
		ConversationID:     normalizedConversationID,
		ProjectKind:        resolveConversationProjectKindLocked(state, normalizedConversationID),
		PendingChangeSetID: "cs_" + randomHex(6),
		Entries:            []ChangeEntry{},
	}
	state.conversationChangeLedgers[normalizedConversationID] = ledger
	return ledger
}

func rebuildConversationChangeLedgerFromStateLocked(state *AppState, conversationID string) *ConversationChangeLedger {
	normalizedConversationID := strings.TrimSpace(conversationID)
	if normalizedConversationID == "" {
		return &ConversationChangeLedger{}
	}
	previous := state.conversationChangeLedgers[normalizedConversationID]
	ledger := &ConversationChangeLedger{
		ConversationID:     normalizedConversationID,
		ProjectKind:        resolveConversationProjectKindLocked(state, normalizedConversationID),
		PendingChangeSetID: "cs_" + randomHex(6),
		Entries:            []ChangeEntry{},
	}
	if previous != nil {
		ledger.LastCommittedCheckpoint = previous.LastCommittedCheckpoint
	}
	for _, executionID := range collectConversationDiffExecutionIDsLocked(state, normalizedConversationID) {
		execution, exists := loadChangeSetExecutionSeedLocked(state, executionID)
		if !exists {
			continue
		}
		for _, item := range state.executionDiffs[executionID] {
			event := ExecutionEvent{
				ExecutionID:    executionID,
				ConversationID: normalizedConversationID,
				QueueIndex:     execution.QueueIndex,
				Timestamp:      execution.UpdatedAt,
			}
			upsertChangeEntryInLedger(ledger, event, execution.MessageID, item)
		}
	}
	state.conversationChangeLedgers[normalizedConversationID] = ledger
	return ledger
}

func upsertChangeEntryInLedger(ledger *ConversationChangeLedger, event ExecutionEvent, messageID string, item DiffItem) {
	if ledger == nil {
		return
	}
	normalizedPath := normalizeDiffPath(item.Path)
	if normalizedPath == "" {
		return
	}
	if strings.TrimSpace(ledger.PendingChangeSetID) == "" {
		ledger.PendingChangeSetID = "cs_" + randomHex(6)
	}
	for index := range ledger.Entries {
		entry := &ledger.Entries[index]
		if entry.Path != normalizedPath {
			continue
		}
		entry.ExecutionID = strings.TrimSpace(event.ExecutionID)
		entry.MessageID = strings.TrimSpace(messageID)
		entry.ChangeType = normalizeDiffChangeType(item.ChangeType)
		entry.Summary = strings.TrimSpace(item.Summary)
		entry.AddedLines = normalizeOptionalDiffLineCount(item.AddedLines)
		entry.DeletedLines = normalizeOptionalDiffLineCount(item.DeletedLines)
		if strings.TrimSpace(item.BeforeBlob) != "" && strings.TrimSpace(entry.BeforeBlob) == "" {
			entry.BeforeBlob = strings.TrimSpace(item.BeforeBlob)
		}
		if strings.TrimSpace(item.AfterBlob) != "" {
			entry.AfterBlob = strings.TrimSpace(item.AfterBlob)
		}
		entry.CreatedAt = firstNonEmpty(strings.TrimSpace(itemTimestamp(item, event.Timestamp)), entry.CreatedAt, time.Now().UTC().Format(time.RFC3339))
		return
	}
	createdAt := itemTimestamp(item, event.Timestamp)
	if strings.TrimSpace(createdAt) == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339)
	}
	ledger.Entries = append(ledger.Entries, ChangeEntry{
		EntryID:      "chg_" + randomHex(6),
		MessageID:    strings.TrimSpace(messageID),
		ExecutionID:  strings.TrimSpace(event.ExecutionID),
		Path:         normalizedPath,
		ChangeType:   normalizeDiffChangeType(item.ChangeType),
		Summary:      strings.TrimSpace(item.Summary),
		AddedLines:   normalizeOptionalDiffLineCount(item.AddedLines),
		DeletedLines: normalizeOptionalDiffLineCount(item.DeletedLines),
		BeforeBlob:   strings.TrimSpace(item.BeforeBlob),
		AfterBlob:    strings.TrimSpace(item.AfterBlob),
		CreatedAt:    createdAt,
	})
}

func itemTimestamp(_ DiffItem, fallback string) string {
	return strings.TrimSpace(fallback)
}

func checkpointFromPayload(payload map[string]any) (CheckpointSummary, bool) {
	checkpointRaw, ok := payload["checkpoint"].(map[string]any)
	if !ok || checkpointRaw == nil {
		return CheckpointSummary{}, false
	}
	checkpoint := CheckpointSummary{
		CheckpointID:  strings.TrimSpace(asStringValue(checkpointRaw["checkpoint_id"])),
		Message:       strings.TrimSpace(asStringValue(checkpointRaw["message"])),
		CreatedAt:     strings.TrimSpace(asStringValue(checkpointRaw["created_at"])),
		GitCommitID:   strings.TrimSpace(asStringValue(checkpointRaw["git_commit_id"])),
		EntriesDigest: strings.TrimSpace(asStringValue(checkpointRaw["entries_digest"])),
	}
	if checkpoint.CheckpointID == "" {
		return CheckpointSummary{}, false
	}
	return checkpoint, true
}

func buildConversationChangeSetLocked(state *AppState, conversationID string) (ConversationChangeSet, error) {
	conversation, exists := loadChangeSetConversationSeedLocked(state, conversationID)
	if !exists {
		return ConversationChangeSet{}, fmt.Errorf("conversation %s not found", conversationID)
	}
	project, exists := loadChangeSetProjectSeedLocked(state, conversation.ProjectID)
	if !exists {
		return ConversationChangeSet{}, fmt.Errorf("project %s not found", conversation.ProjectID)
	}
	ledger := ensureConversationChangeLedgerLocked(state, conversationID)
	if strings.TrimSpace(ledger.ProjectKind) == "" {
		ledger.ProjectKind = resolveProjectKind(project)
	}
	entries := cloneChangeEntries(ledger.Entries)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Path == entries[j].Path {
			return entries[i].CreatedAt < entries[j].CreatedAt
		}
		return entries[i].Path < entries[j].Path
	})
	addedLines := 0
	deletedLines := 0
	for _, entry := range entries {
		if entry.AddedLines != nil {
			addedLines += *entry.AddedLines
		}
		if entry.DeletedLines != nil {
			deletedLines += *entry.DeletedLines
		}
	}
	busy := hasMutableExecutionsLocked(state, conversationID)
	capability := ChangeSetCapability{
		CanCommit:  !busy,
		CanDiscard: !busy,
		CanExport:  true,
	}
	if busy {
		capability.Reason = "Pending or running executions exist; wait until conversation is idle"
	}
	if len(entries) == 0 {
		capability.CanCommit = false
		capability.CanDiscard = false
		if capability.Reason == "" {
			capability.Reason = "No pending changes in current conversation"
		}
	}
	return ConversationChangeSet{
		ChangeSetID:             firstNonEmpty(strings.TrimSpace(ledger.PendingChangeSetID), "cs_"+randomHex(6)),
		ConversationID:          conversationID,
		ProjectKind:             resolveProjectKind(project),
		Entries:                 entries,
		FileCount:               len(entries),
		AddedLines:              addedLines,
		DeletedLines:            deletedLines,
		Capability:              capability,
		SuggestedMessage:        CommitSuggestion{Message: suggestCommitMessage(entries, resolveProjectKind(project))},
		LastCommittedCheckpoint: cloneCheckpointSummary(ledger.LastCommittedCheckpoint),
	}, nil
}

func cloneCheckpointSummary(input *CheckpointSummary) *CheckpointSummary {
	if input == nil {
		return nil
	}
	copyValue := *input
	return &copyValue
}

func hasMutableExecutionsLocked(state *AppState, conversationID string) bool {
	for _, executionID := range state.conversationExecutionOrder[conversationID] {
		execution, exists := loadChangeSetExecutionSeedLocked(state, executionID)
		if !exists {
			continue
		}
		switch execution.State {
		case RunStateQueued, RunStatePending, RunStateExecuting, RunStateConfirming, RunStateAwaitingInput:
			return true
		}
	}
	for _, run := range listChangeSetRuntimeRunsByConversationLocked(state, conversationID) {
		switch RunState(strings.TrimSpace(run.State)) {
		case RunStateQueued, RunStatePending, RunStateExecuting, RunStateConfirming, RunStateAwaitingInput:
			return true
		}
	}
	return false
}

func loadChangeSetExecutionSeedLocked(state *AppState, executionID string) (Execution, bool) {
	normalizedExecutionID := strings.TrimSpace(executionID)
	if state == nil || normalizedExecutionID == "" {
		return Execution{}, false
	}
	if execution, exists := state.executions[normalizedExecutionID]; exists {
		return execution, true
	}
	service, ok := newExecutionQueryService(state)
	if !ok {
		return Execution{}, false
	}
	item, exists, err := service.repositories.Runs.GetByID(context.Background(), normalizedExecutionID)
	if err != nil || !exists {
		return Execution{}, false
	}
	execution := toExecutionFromRuntimeRun(item)
	assignQueueIndexFromConversationOrderLocked(state, &execution)
	state.executions[normalizedExecutionID] = execution
	return execution, true
}

func loadChangeSetConversationSeed(ctx context.Context, state *AppState, conversationID string) (Conversation, bool) {
	normalizedConversationID := strings.TrimSpace(conversationID)
	if state == nil || normalizedConversationID == "" {
		return Conversation{}, false
	}

	state.mu.RLock()
	conversation, exists := state.conversations[normalizedConversationID]
	state.mu.RUnlock()
	if exists {
		return conversation, true
	}

	service, ok := newExecutionQueryService(state)
	if !ok {
		return Conversation{}, false
	}
	item, exists, err := service.repositories.Sessions.GetByID(ctx, normalizedConversationID)
	if err != nil {
		log.Printf("runtime v1 changeset conversation lookup failed, fallback to in-memory map: %v", err)
		return Conversation{}, false
	}
	if !exists {
		return Conversation{}, false
	}
	return toConversationFromRuntimeSessionRecord(item), true
}

func loadChangeSetConversationSeedLocked(state *AppState, conversationID string) (Conversation, bool) {
	normalizedConversationID := strings.TrimSpace(conversationID)
	if state == nil || normalizedConversationID == "" {
		return Conversation{}, false
	}
	if conversation, exists := state.conversations[normalizedConversationID]; exists {
		return conversation, true
	}
	service, ok := newExecutionQueryService(state)
	if !ok {
		return Conversation{}, false
	}
	item, exists, err := service.repositories.Sessions.GetByID(context.Background(), normalizedConversationID)
	if err != nil || !exists {
		return Conversation{}, false
	}
	seed := toConversationFromRuntimeSessionRecord(item)
	state.conversations[normalizedConversationID] = seed
	return seed, true
}

func loadChangeSetProjectSeedLocked(state *AppState, projectID string) (Project, bool) {
	normalizedProjectID := strings.TrimSpace(projectID)
	if state == nil || normalizedProjectID == "" {
		return Project{}, false
	}
	if project, exists := state.projects[normalizedProjectID]; exists {
		return project, true
	}
	if state.authz == nil {
		return Project{}, false
	}
	item, exists, err := state.authz.getProject(normalizedProjectID)
	if err != nil || !exists {
		return Project{}, false
	}
	state.projects[normalizedProjectID] = item
	return item, true
}

func toConversationFromRuntimeSessionRecord(item RuntimeSessionRecord) Conversation {
	defaultMode := NormalizePermissionMode(item.DefaultMode)
	seed := Conversation{
		ID:            item.ID,
		WorkspaceID:   item.WorkspaceID,
		ProjectID:     item.ProjectID,
		Name:          item.Name,
		QueueState:    QueueStateIdle,
		DefaultMode:   defaultMode,
		ModelConfigID: item.ModelConfigID,
		RuleIDs:       append([]string{}, item.RuleIDs...),
		SkillIDs:      append([]string{}, item.SkillIDs...),
		MCPIDs:        append([]string{}, item.MCPIDs...),
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
	if item.ActiveRunID != nil {
		activeRunID := strings.TrimSpace(*item.ActiveRunID)
		if activeRunID != "" {
			seed.ActiveExecutionID = &activeRunID
			seed.QueueState = QueueStateRunning
		}
	}
	return seed
}

func ensureChangeSetContextSeedsLocked(state *AppState, conversationSeed Conversation, projectSeed Project) (Conversation, Project, bool) {
	if state == nil {
		return Conversation{}, Project{}, false
	}
	conversationID := strings.TrimSpace(conversationSeed.ID)
	if conversationID == "" {
		return Conversation{}, Project{}, false
	}
	conversation, exists := state.conversations[conversationID]
	if !exists {
		conversation = conversationSeed
		state.conversations[conversationID] = conversation
	}
	projectID := strings.TrimSpace(conversation.ProjectID)
	if projectID == "" {
		return conversation, Project{}, false
	}
	project, projectExists := state.projects[projectID]
	if projectExists {
		return conversation, project, true
	}
	if strings.TrimSpace(projectSeed.ID) == projectID {
		state.projects[projectID] = projectSeed
		return conversation, projectSeed, true
	}
	projectLoaded, loaded := loadChangeSetProjectSeedLocked(state, projectID)
	if !loaded {
		return conversation, Project{}, false
	}
	return conversation, projectLoaded, true
}

func listChangeSetRuntimeRunsByConversationLocked(state *AppState, conversationID string) []RuntimeRunRecord {
	normalizedConversationID := strings.TrimSpace(conversationID)
	if state == nil || normalizedConversationID == "" {
		return []RuntimeRunRecord{}
	}
	service, ok := newExecutionQueryService(state)
	if !ok {
		return []RuntimeRunRecord{}
	}
	items := []RuntimeRunRecord{}
	offset := 0
	for {
		page, err := service.repositories.Runs.ListBySession(context.Background(), normalizedConversationID, RepositoryPage{
			Limit:  maxRepositoryPageLimit,
			Offset: offset,
		})
		if err != nil {
			return items
		}
		if len(page) == 0 {
			break
		}
		items = append(items, page...)
		if len(page) < maxRepositoryPageLimit {
			break
		}
		offset += len(page)
	}
	return items
}

func resolveConversationProjectKindLocked(state *AppState, conversationID string) string {
	conversation, exists := loadChangeSetConversationSeedLocked(state, conversationID)
	if !exists {
		return "non_git"
	}
	project, exists := loadChangeSetProjectSeedLocked(state, conversation.ProjectID)
	if !exists {
		return "non_git"
	}
	return resolveProjectKind(project)
}

func resolveProjectKind(project Project) string {
	if project.IsGit && isGitRepositoryPath(project.RepoPath) {
		return "git"
	}
	return "non_git"
}

func bumpPendingChangeSetIDLocked(ledger *ConversationChangeLedger) {
	if ledger == nil {
		return
	}
	ledger.PendingChangeSetID = "cs_" + randomHex(6)
}

func suggestCommitMessage(entries []ChangeEntry, projectKind string) string {
	if len(entries) == 0 {
		if projectKind == "git" {
			return "chore: no-op changeset"
		}
		return "checkpoint: no-op changeset"
	}
	addedFiles := 0
	modifiedFiles := 0
	deletedFiles := 0
	topDirs := map[string]struct{}{}
	for _, entry := range entries {
		switch normalizeDiffChangeType(entry.ChangeType) {
		case "added":
			addedFiles += 1
		case "deleted":
			deletedFiles += 1
		default:
			modifiedFiles += 1
		}
		firstSegment := strings.Split(strings.TrimPrefix(entry.Path, "/"), "/")[0]
		if strings.TrimSpace(firstSegment) != "" {
			topDirs[firstSegment] = struct{}{}
		}
	}
	dirs := make([]string, 0, len(topDirs))
	for dir := range topDirs {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	scope := "workspace"
	if len(dirs) > 0 {
		scope = strings.Join(dirs, ",")
	}
	verb := "update"
	if addedFiles > 0 && modifiedFiles == 0 && deletedFiles == 0 {
		verb = "add"
	}
	if deletedFiles > 0 && addedFiles == 0 && modifiedFiles == 0 {
		verb = "remove"
	}
	prefix := "chore"
	if projectKind == "non_git" {
		prefix = "checkpoint"
	}
	return fmt.Sprintf("%s(%s): %s %d files (add %d / mod %d / del %d)", prefix, scope, verb, len(entries), addedFiles, modifiedFiles, deletedFiles)
}

func cloneChangeEntries(items []ChangeEntry) []ChangeEntry {
	result := make([]ChangeEntry, 0, len(items))
	for _, item := range items {
		copyItem := item
		copyItem.Path = strings.TrimSpace(item.Path)
		result = append(result, copyItem)
	}
	return result
}

func uniqueEntryPaths(entries []ChangeEntry) []string {
	unique := map[string]struct{}{}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		path := normalizeDiffPath(entry.Path)
		if path == "" {
			continue
		}
		if _, exists := unique[path]; exists {
			continue
		}
		unique[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func digestChangeEntries(entries []ChangeEntry) string {
	if len(entries) == 0 {
		return ""
	}
	payload := make([]string, 0, len(entries))
	for _, entry := range entries {
		added := ""
		if entry.AddedLines != nil {
			added = fmt.Sprintf("%d", *entry.AddedLines)
		}
		deleted := ""
		if entry.DeletedLines != nil {
			deleted = fmt.Sprintf("%d", *entry.DeletedLines)
		}
		payload = append(payload, fmt.Sprintf("%s|%s|%s|%s", entry.Path, entry.ChangeType, added, deleted))
	}
	sort.Strings(payload)
	hash := sha256.Sum256([]byte(strings.Join(payload, "\n")))
	return hex.EncodeToString(hash[:])
}

func restoreNonGitWorkingTreePaths(projectPath string, entries []ChangeEntry) error {
	for _, entry := range entries {
		path := normalizeDiffPath(entry.Path)
		if path == "" {
			continue
		}
		targetPath, err := resolveProjectRelativePath(projectPath, path)
		if err != nil {
			return err
		}
		beforeBlob := strings.TrimSpace(entry.BeforeBlob)
		if normalizeDiffChangeType(entry.ChangeType) == "added" && beforeBlob == "" {
			if err := os.RemoveAll(targetPath); err != nil {
				return err
			}
			continue
		}
		if beforeBlob == "" {
			if normalizeDiffChangeType(entry.ChangeType) == "deleted" {
				continue
			}
			if err := os.RemoveAll(targetPath); err != nil {
				return err
			}
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(beforeBlob)
		if err != nil {
			return fmt.Errorf("decode before_blob for %s failed: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, decoded, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func renderChangeSetFilesArchiveBase64(projectPath string, entries []ChangeEntry) (string, error) {
	if strings.TrimSpace(projectPath) == "" {
		return "", errors.New("project path is empty")
	}
	buffer := bytes.NewBuffer(nil)
	writer := zip.NewWriter(buffer)
	manifestLines := make([]string, 0)
	paths := uniqueEntryPaths(entries)
	entryByPath := map[string]ChangeEntry{}
	for _, entry := range entries {
		path := normalizeDiffPath(entry.Path)
		if path == "" {
			continue
		}
		entryByPath[path] = entry
	}
	for _, path := range paths {
		entry := entryByPath[path]
		if normalizeDiffChangeType(entry.ChangeType) == "deleted" {
			manifestLines = append(manifestLines, fmt.Sprintf("Deleted: %s", path))
			continue
		}
		targetPath, err := resolveProjectRelativePath(projectPath, path)
		if err != nil {
			manifestLines = append(manifestLines, fmt.Sprintf("Skip %s: invalid path", path))
			continue
		}
		info, err := os.Stat(targetPath)
		if err != nil {
			manifestLines = append(manifestLines, fmt.Sprintf("Missing %s: %v", path, err))
			continue
		}
		if info.IsDir() {
			manifestLines = append(manifestLines, fmt.Sprintf("Skip %s: directory is not exported", path))
			continue
		}
		content, err := os.ReadFile(targetPath)
		if err != nil {
			manifestLines = append(manifestLines, fmt.Sprintf("Read failed %s: %v", path, err))
			continue
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			manifestLines = append(manifestLines, fmt.Sprintf("Header failed %s: %v", path, err))
			continue
		}
		header.Method = zip.Deflate
		header.Name = normalizeDiffPath(path)
		entryWriter, err := writer.CreateHeader(header)
		if err != nil {
			manifestLines = append(manifestLines, fmt.Sprintf("Zip failed %s: %v", path, err))
			continue
		}
		if _, err := entryWriter.Write(content); err != nil {
			manifestLines = append(manifestLines, fmt.Sprintf("Write failed %s: %v", path, err))
			continue
		}
	}
	if len(paths) == 0 {
		manifestLines = append(manifestLines, "No pending changes in current conversation.")
	}
	if len(manifestLines) > 0 {
		manifestWriter, err := writer.Create("_goyais_export_manifest.txt")
		if err != nil {
			_ = writer.Close()
			return "", err
		}
		manifestContent := strings.Join(manifestLines, "\n") + "\n"
		if _, err := manifestWriter.Write([]byte(manifestContent)); err != nil {
			_ = writer.Close()
			return "", err
		}
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}
