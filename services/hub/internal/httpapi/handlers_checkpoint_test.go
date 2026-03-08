package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

type checkpointApplicationServiceStub struct {
	listCalled    bool
	listSessionID string
	listItems     []Checkpoint
	listErr       error

	createCalled    bool
	createSessionID string
	createMessage   string
	createResult    Checkpoint
	createErr       error

	rollbackCalled       bool
	rollbackSessionID    string
	rollbackCheckpointID string
	rollbackCheckpoint   Checkpoint
	rollbackSession      Conversation
	rollbackErr          error
}

func (s *checkpointApplicationServiceStub) ListSessionCheckpoints(_ context.Context, sessionID string) ([]Checkpoint, error) {
	s.listCalled = true
	s.listSessionID = sessionID
	return append([]Checkpoint{}, s.listItems...), s.listErr
}

func (s *checkpointApplicationServiceStub) CreateSessionCheckpoint(_ context.Context, sessionID string, message string) (Checkpoint, error) {
	s.createCalled = true
	s.createSessionID = sessionID
	s.createMessage = message
	return s.createResult, s.createErr
}

func (s *checkpointApplicationServiceStub) RollbackSessionToCheckpoint(_ context.Context, sessionID string, checkpointID string) (Checkpoint, Conversation, error) {
	s.rollbackCalled = true
	s.rollbackSessionID = sessionID
	s.rollbackCheckpointID = checkpointID
	return s.rollbackCheckpoint, s.rollbackSession, s.rollbackErr
}

func TestCheckpointEndpointsCreateListAndRollbackSessionState(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Checkpoint", "http://127.0.0.1:9140", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "checkpoint_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": t.TempDir(),
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	project := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &project)
	projectID := project["id"].(string)

	modelConfigID := createModelResourceConfigForTest(t, router, workspaceID, authHeaders, "OpenAI", "gpt-5.3")
	bindProjectConfigWithModelForTest(t, router, projectID, modelConfigID, authHeaders)

	sessionRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/sessions", map[string]any{
		"name": "Checkpoint Session",
	}, authHeaders)
	if sessionRes.Code != http.StatusCreated {
		t.Fatalf("expected create session 201, got %d (%s)", sessionRes.Code, sessionRes.Body.String())
	}
	sessionPayload := map[string]any{}
	mustDecodeJSON(t, sessionRes.Body.Bytes(), &sessionPayload)
	sessionID := sessionPayload["id"].(string)

	firstRunRes := performJSONRequest(t, router, http.MethodPost, "/v1/sessions/"+sessionID+"/runs", map[string]any{
		"raw_input":       "first prompt",
		"mode":            "default",
		"model_config_id": modelConfigID,
	}, authHeaders)
	if firstRunRes.Code != http.StatusCreated {
		t.Fatalf("expected first run 201, got %d (%s)", firstRunRes.Code, firstRunRes.Body.String())
	}

	checkpointCreateRes := performJSONRequest(t, router, http.MethodPost, "/v1/sessions/"+sessionID+"/checkpoints", map[string]any{
		"message": "before second prompt",
	}, authHeaders)
	if checkpointCreateRes.Code != http.StatusCreated {
		t.Fatalf("expected checkpoint create 201, got %d (%s)", checkpointCreateRes.Code, checkpointCreateRes.Body.String())
	}
	checkpointPayload := map[string]any{}
	mustDecodeJSON(t, checkpointCreateRes.Body.Bytes(), &checkpointPayload)
	checkpointID := asString(checkpointPayload["checkpoint_id"])
	if checkpointID == "" {
		t.Fatalf("expected checkpoint_id in create response")
	}

	checkpointListRes := performJSONRequest(t, router, http.MethodGet, "/v1/sessions/"+sessionID+"/checkpoints", nil, authHeaders)
	if checkpointListRes.Code != http.StatusOK {
		t.Fatalf("expected checkpoint list 200, got %d (%s)", checkpointListRes.Code, checkpointListRes.Body.String())
	}
	checkpointListPayload := map[string]any{}
	mustDecodeJSON(t, checkpointListRes.Body.Bytes(), &checkpointListPayload)
	items, ok := checkpointListPayload["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one checkpoint in list, got %#v", checkpointListPayload["items"])
	}

	secondRunRes := performJSONRequest(t, router, http.MethodPost, "/v1/sessions/"+sessionID+"/runs", map[string]any{
		"raw_input":       "second prompt",
		"mode":            "default",
		"model_config_id": modelConfigID,
	}, authHeaders)
	if secondRunRes.Code != http.StatusCreated {
		t.Fatalf("expected second run 201, got %d (%s)", secondRunRes.Code, secondRunRes.Body.String())
	}

	rollbackRes := performJSONRequest(t, router, http.MethodPost, "/v1/sessions/"+sessionID+"/checkpoints/"+checkpointID+"/rollback", map[string]any{}, authHeaders)
	if rollbackRes.Code != http.StatusOK {
		t.Fatalf("expected checkpoint rollback 200, got %d (%s)", rollbackRes.Code, rollbackRes.Body.String())
	}
	rollbackPayload := map[string]any{}
	mustDecodeJSON(t, rollbackRes.Body.Bytes(), &rollbackPayload)
	if rollbackPayload["ok"] != true {
		t.Fatalf("expected rollback ok=true, got %#v", rollbackPayload["ok"])
	}
	restoredSession, ok := rollbackPayload["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected rollback session payload, got %#v", rollbackPayload["session"])
	}
	if asString(restoredSession["project_id"]) != projectID || asString(restoredSession["name"]) != "Checkpoint Session" {
		t.Fatalf("expected rollback session details preserved, got %#v", restoredSession)
	}

	detailRes := performJSONRequest(t, router, http.MethodGet, "/v1/sessions/"+sessionID, nil, authHeaders)
	if detailRes.Code != http.StatusOK {
		t.Fatalf("expected session detail 200, got %d (%s)", detailRes.Code, detailRes.Body.String())
	}
	detailPayload := map[string]any{}
	mustDecodeJSON(t, detailRes.Body.Bytes(), &detailPayload)
	messages, ok := detailPayload["messages"].([]any)
	if !ok {
		t.Fatalf("expected messages array, got %#v", detailPayload["messages"])
	}
	if len(messages) != 1 {
		t.Fatalf("expected checkpoint rollback to restore one message, got %d", len(messages))
	}
}

func TestCheckpointHandlersRecordBusinessAuditEntries(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_checkpoint_audit"
	sessionID := "conv_checkpoint_audit"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Checkpoint Audit Project",
		RepoPath:    t.TempDir(),
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.conversations[sessionID] = Conversation{
		ID:            sessionID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Checkpoint Audit Session",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_checkpoint_audit",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationMessages[sessionID] = []ConversationMessage{{
		ID:             "msg_checkpoint_audit",
		ConversationID: sessionID,
		Role:           MessageRoleUser,
		Content:        "checkpoint me",
		CreatedAt:      now,
	}}
	state.adminAudit = nil

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sessions/{session_id}/checkpoints", SessionCheckpointsHandler(state))
	mux.HandleFunc("/v1/sessions/{session_id}/checkpoints/{checkpoint_id}/rollback", SessionCheckpointRollbackHandler(state))

	createRes := performJSONRequest(t, mux, http.MethodPost, "/v1/sessions/"+sessionID+"/checkpoints", map[string]any{
		"message": "savepoint",
	}, nil)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d (%s)", createRes.Code, createRes.Body.String())
	}
	checkpoint := Checkpoint{}
	mustDecodeJSON(t, createRes.Body.Bytes(), &checkpoint)

	listRes := performJSONRequest(t, mux, http.MethodGet, "/v1/sessions/"+sessionID+"/checkpoints", nil, nil)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d (%s)", listRes.Code, listRes.Body.String())
	}

	rollbackRes := performJSONRequest(t, mux, http.MethodPost, "/v1/sessions/"+sessionID+"/checkpoints/"+checkpoint.CheckpointID+"/rollback", map[string]any{}, nil)
	if rollbackRes.Code != http.StatusOK {
		t.Fatalf("expected rollback 200, got %d (%s)", rollbackRes.Code, rollbackRes.Body.String())
	}

	state.mu.RLock()
	defer state.mu.RUnlock()
	actionCounts := map[string]int{}
	for _, entry := range state.adminAudit {
		actionCounts[entry.Action]++
	}
	if actionCounts["session.write"] == 0 {
		t.Fatalf("expected checkpoint create audit entry, got %#v", state.adminAudit)
	}
	if actionCounts["session.read"] == 0 {
		t.Fatalf("expected checkpoint list audit entry, got %#v", state.adminAudit)
	}
	if actionCounts["run.control"] == 0 {
		t.Fatalf("expected checkpoint rollback audit entry, got %#v", state.adminAudit)
	}
}

func TestCheckpointRollbackAuthorizationAuditUsesSessionIDTarget(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_checkpoint_authz_target"
	sessionID := "conv_checkpoint_authz_target"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Checkpoint Target Project",
		RepoPath:    t.TempDir(),
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.conversations[sessionID] = Conversation{
		ID:            sessionID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Checkpoint Target Session",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_checkpoint_target",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationMessages[sessionID] = []ConversationMessage{{
		ID:             "msg_checkpoint_target",
		ConversationID: sessionID,
		Role:           MessageRoleUser,
		Content:        "checkpoint me",
		CreatedAt:      now,
	}}
	state.adminAudit = nil

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sessions/{session_id}/checkpoints", SessionCheckpointsHandler(state))
	mux.HandleFunc("/v1/sessions/{session_id}/checkpoints/{checkpoint_id}/rollback", SessionCheckpointRollbackHandler(state))

	createRes := performJSONRequest(t, mux, http.MethodPost, "/v1/sessions/"+sessionID+"/checkpoints", map[string]any{
		"message": "savepoint",
	}, nil)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d (%s)", createRes.Code, createRes.Body.String())
	}
	checkpoint := Checkpoint{}
	mustDecodeJSON(t, createRes.Body.Bytes(), &checkpoint)

	rollbackRes := performJSONRequest(t, mux, http.MethodPost, "/v1/sessions/"+sessionID+"/checkpoints/"+checkpoint.CheckpointID+"/rollback", map[string]any{}, nil)
	if rollbackRes.Code != http.StatusOK {
		t.Fatalf("expected rollback 200, got %d (%s)", rollbackRes.Code, rollbackRes.Body.String())
	}

	state.mu.RLock()
	defer state.mu.RUnlock()
	for _, entry := range state.adminAudit {
		if entry.Action != "authz.run.control" {
			continue
		}
		if entry.Resource != sessionID {
			t.Fatalf("expected authz.run.control resource %q, got %#v", sessionID, entry)
		}
		return
	}
	t.Fatalf("expected authz.run.control audit entry, got %#v", state.adminAudit)
}

func TestSessionCheckpointsHandlerUsesCheckpointServiceWhenConfigured(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_checkpoint_service"
	sessionID := "conv_checkpoint_service"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Checkpoint Service Project",
		RepoPath:    t.TempDir(),
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.conversations[sessionID] = Conversation{
		ID:            sessionID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Checkpoint Service Session",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_checkpoint_service",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	state.checkpointService = &checkpointApplicationServiceStub{
		listItems: []Checkpoint{{
			CheckpointSummary: CheckpointSummary{
				CheckpointID: "cp_stub_list",
				Message:      "from service list",
				CreatedAt:    now,
			},
			SessionID: sessionID,
			Session:   &Conversation{ID: sessionID},
		}},
		createResult: Checkpoint{
			CheckpointSummary: CheckpointSummary{
				CheckpointID: "cp_stub_create",
				Message:      "from service create",
				CreatedAt:    now,
			},
			SessionID: sessionID,
			Session:   &Conversation{ID: sessionID},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sessions/{session_id}/checkpoints", SessionCheckpointsHandler(state))

	listRes := performJSONRequest(t, mux, http.MethodGet, "/v1/sessions/"+sessionID+"/checkpoints", nil, nil)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d (%s)", listRes.Code, listRes.Body.String())
	}
	createRes := performJSONRequest(t, mux, http.MethodPost, "/v1/sessions/"+sessionID+"/checkpoints", map[string]any{
		"message": "create via service",
	}, nil)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d (%s)", createRes.Code, createRes.Body.String())
	}

	stub := state.checkpointService.(*checkpointApplicationServiceStub)
	if !stub.listCalled || stub.listSessionID != sessionID {
		t.Fatalf("expected list to delegate to checkpoint service, got %#v", stub)
	}
	if !stub.createCalled || stub.createSessionID != sessionID || stub.createMessage != "create via service" {
		t.Fatalf("expected create to delegate to checkpoint service, got %#v", stub)
	}
	if listRes.Body.String() == "" || createRes.Body.String() == "" {
		t.Fatalf("expected non-empty service-backed responses")
	}
}

func TestSessionCheckpointRollbackHandlerUsesCheckpointServiceWhenConfigured(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_checkpoint_service_rollback"
	sessionID := "conv_checkpoint_service_rollback"
	checkpointID := "cp_stub_rollback"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Checkpoint Service Rollback Project",
		RepoPath:    t.TempDir(),
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.conversations[sessionID] = Conversation{
		ID:            sessionID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Checkpoint Service Rollback Session",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_checkpoint_service",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	state.checkpointService = &checkpointApplicationServiceStub{
		rollbackCheckpoint: Checkpoint{
			CheckpointSummary: CheckpointSummary{
				CheckpointID: checkpointID,
				Message:      "rollback via service",
				CreatedAt:    now,
			},
			SessionID: sessionID,
			Session:   &Conversation{ID: sessionID},
		},
		rollbackSession: Conversation{
			ID:            sessionID,
			WorkspaceID:   localWorkspaceID,
			ProjectID:     projectID,
			Name:          "Restored by service",
			QueueState:    QueueStateIdle,
			DefaultMode:   PermissionModeDefault,
			ModelConfigID: "rc_model_checkpoint_service",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sessions/{session_id}/checkpoints/{checkpoint_id}/rollback", SessionCheckpointRollbackHandler(state))

	rollbackRes := performJSONRequest(t, mux, http.MethodPost, "/v1/sessions/"+sessionID+"/checkpoints/"+checkpointID+"/rollback", map[string]any{}, nil)
	if rollbackRes.Code != http.StatusOK {
		t.Fatalf("expected rollback 200, got %d (%s)", rollbackRes.Code, rollbackRes.Body.String())
	}

	stub := state.checkpointService.(*checkpointApplicationServiceStub)
	if !stub.rollbackCalled || stub.rollbackSessionID != sessionID || stub.rollbackCheckpointID != checkpointID {
		t.Fatalf("expected rollback to delegate to checkpoint service, got %#v", stub)
	}
	if !strings.Contains(rollbackRes.Body.String(), checkpointID) {
		t.Fatalf("expected rollback response to include checkpoint id, got %s", rollbackRes.Body.String())
	}
}
