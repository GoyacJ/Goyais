package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appcommands "goyais/services/hub/internal/application/commands"
	appqueries "goyais/services/hub/internal/application/queries"
)

type sessionQueryApplicationStub struct {
	listSessionsResp    []appqueries.Session
	listSessionsNext    *string
	listSessionsCalled  bool
	sessionDetailResp   appqueries.SessionDetail
	sessionDetailExists bool
	sessionDetailCalled bool
	runEventsResp       []appqueries.RunEvent
	runEventsCalled     bool
}

func (s *sessionQueryApplicationStub) ListSessions(_ context.Context, _ appqueries.ListSessionsRequest) ([]appqueries.Session, *string, error) {
	s.listSessionsCalled = true
	return append([]appqueries.Session{}, s.listSessionsResp...), s.listSessionsNext, nil
}

func (s *sessionQueryApplicationStub) GetSessionDetail(_ context.Context, _ string) (appqueries.SessionDetail, bool, error) {
	s.sessionDetailCalled = true
	return s.sessionDetailResp, s.sessionDetailExists, nil
}

func (s *sessionQueryApplicationStub) GetRunEvents(_ context.Context, _ appqueries.GetRunEventsRequest) ([]appqueries.RunEvent, error) {
	s.runEventsCalled = true
	return append([]appqueries.RunEvent{}, s.runEventsResp...), nil
}

type sessionCommandApplicationStub struct {
	createResult appcommands.CreateSessionResult
	createErr    error
	createCalled bool
	createCmd    appcommands.CreateSessionCommand
	onCreate     func(appcommands.CreateSessionCommand)

	submitResult appcommands.SubmitMessageResult
	submitErr    error
	submitCalled bool
	submitCmd    appcommands.SubmitMessageCommand
	onSubmit     func(appcommands.SubmitMessageCommand)

	controlResult appcommands.ControlRunResult
	controlErr    error
	controlCalled bool
	controlCmd    appcommands.ControlRunCommand
	onControl     func(appcommands.ControlRunCommand)
}

func (s *sessionCommandApplicationStub) CreateSession(_ context.Context, cmd appcommands.CreateSessionCommand) (appcommands.CreateSessionResult, error) {
	s.createCalled = true
	s.createCmd = cmd
	if s.onCreate != nil {
		s.onCreate(cmd)
	}
	return s.createResult, s.createErr
}

func (s *sessionCommandApplicationStub) SubmitMessage(_ context.Context, cmd appcommands.SubmitMessageCommand) (appcommands.SubmitMessageResult, error) {
	s.submitCalled = true
	s.submitCmd = cmd
	if s.onSubmit != nil {
		s.onSubmit(cmd)
	}
	return s.submitResult, s.submitErr
}

func (s *sessionCommandApplicationStub) ControlRun(_ context.Context, cmd appcommands.ControlRunCommand) (appcommands.ControlRunResult, error) {
	s.controlCalled = true
	s.controlCmd = cmd
	if s.onControl != nil {
		s.onControl(cmd)
	}
	return s.controlResult, s.controlErr
}

func TestConversationsHandlerUsesCQRSQueryServiceWhenFeatureEnabled(t *testing.T) {
	t.Setenv("FEATURE_CQRS", "true")
	state := NewAppState(nil)
	state.sessionQueries = &sessionQueryApplicationStub{
		listSessionsResp: []appqueries.Session{
			{
				ID:          "conv_cqrs",
				WorkspaceID: localWorkspaceID,
				ProjectID:   "proj_cqrs",
				Name:        "From CQRS",
				QueueState:  string(QueueStateIdle),
				DefaultMode: string(PermissionModeDefault),
				CreatedAt:   "2026-03-08T00:00:00Z",
				UpdatedAt:   "2026-03-08T00:00:00Z",
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions?workspace_id="+localWorkspaceID, nil)
	recorder := httptest.NewRecorder()
	ConversationsHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	if !state.sessionQueries.(*sessionQueryApplicationStub).listSessionsCalled {
		t.Fatalf("expected CQRS list query service to be called")
	}
	if !strings.Contains(recorder.Body.String(), "From CQRS") {
		t.Fatalf("expected response to contain CQRS session payload, got %s", recorder.Body.String())
	}
}

func TestConversationByIDHandlerUsesCQRSQueryServiceWhenFeatureEnabled(t *testing.T) {
	t.Setenv("FEATURE_CQRS", "true")
	state := NewAppState(nil)
	state.sessionQueries = &sessionQueryApplicationStub{
		sessionDetailResp: appqueries.SessionDetail{
			Session: appqueries.Session{
				ID:          "conv_detail_cqrs",
				WorkspaceID: localWorkspaceID,
				ProjectID:   "proj_detail_cqrs",
				Name:        "CQRS Detail",
				QueueState:  string(QueueStateIdle),
				DefaultMode: string(PermissionModeDefault),
				CreatedAt:   "2026-03-08T00:00:00Z",
				UpdatedAt:   "2026-03-08T00:00:00Z",
			},
			Messages: []appqueries.SessionMessage{
				{
					ID:        "msg_cqrs",
					SessionID: "conv_detail_cqrs",
					Role:      string(MessageRoleUser),
					Content:   "hello from cqrs",
					CreatedAt: "2026-03-08T00:00:00Z",
				},
			},
		},
		sessionDetailExists: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/conv_detail_cqrs", nil)
	req.SetPathValue("session_id", "conv_detail_cqrs")
	recorder := httptest.NewRecorder()
	ConversationByIDHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	if !state.sessionQueries.(*sessionQueryApplicationStub).sessionDetailCalled {
		t.Fatalf("expected CQRS detail query service to be called")
	}
	if !strings.Contains(recorder.Body.String(), "CQRS Detail") {
		t.Fatalf("expected response to contain CQRS detail payload, got %s", recorder.Body.String())
	}
}

func TestProjectConversationsHandlerUsesCQRSCommandServiceWhenFeatureEnabled(t *testing.T) {
	t.Setenv("FEATURE_CQRS", "true")
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_create_cqrs"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "CQRS Create",
		RepoPath:    "/tmp/cqrs-create",
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	state.sessionCommands = &sessionCommandApplicationStub{
		createResult: appcommands.CreateSessionResult{SessionID: "conv_create_cqrs"},
		onCreate: func(cmd appcommands.CreateSessionCommand) {
			state.conversations["conv_create_cqrs"] = Conversation{
				ID:            "conv_create_cqrs",
				WorkspaceID:   cmd.WorkspaceID,
				ProjectID:     cmd.ProjectID,
				Name:          cmd.Name,
				QueueState:    QueueStateIdle,
				DefaultMode:   PermissionModeDefault,
				ModelConfigID: "rc_model_create_cqrs",
				CreatedAt:     now,
				UpdatedAt:     now,
			}
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/projects/"+projectID+"/sessions", strings.NewReader(`{"name":"Created from CQRS"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("project_id", projectID)
	recorder := httptest.NewRecorder()

	ProjectConversationsHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	stub := state.sessionCommands.(*sessionCommandApplicationStub)
	if !stub.createCalled {
		t.Fatalf("expected CQRS create command service to be called")
	}
	if stub.createCmd.ProjectID != projectID {
		t.Fatalf("expected project_id %s, got %#v", projectID, stub.createCmd)
	}
	if stub.createCmd.WorkspaceID != localWorkspaceID {
		t.Fatalf("expected workspace_id %s, got %#v", localWorkspaceID, stub.createCmd)
	}
	if stub.createCmd.Name != "Created from CQRS" {
		t.Fatalf("expected name propagated, got %#v", stub.createCmd)
	}
	if !strings.Contains(recorder.Body.String(), "conv_create_cqrs") {
		t.Fatalf("expected response to contain created session id, got %s", recorder.Body.String())
	}
}

func TestProjectConversationsHandlerCreatesSessionViaCQRSCommandAdapter(t *testing.T) {
	t.Setenv("FEATURE_CQRS", "true")
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_create_cqrs_real"
	modelConfigID := "rc_model_create_cqrs_real"

	state.projects[projectID] = Project{
		ID:                   projectID,
		WorkspaceID:          localWorkspaceID,
		Name:                 "CQRS Create Real",
		RepoPath:             "/tmp/cqrs-create-real",
		DefaultModelConfigID: modelConfigID,
		DefaultMode:          PermissionModeDefault,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	state.projectConfigs[projectID] = ProjectConfig{
		ProjectID:            projectID,
		ModelConfigIDs:       []string{modelConfigID},
		DefaultModelConfigID: &modelConfigID,
		UpdatedAt:            now,
	}
	mustSaveTestResourceConfig(t, state, ResourceConfig{
		ID:          modelConfigID,
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-5.3",
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/projects/"+projectID+"/sessions", strings.NewReader(`{"name":"Created by adapter"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("project_id", projectID)
	recorder := httptest.NewRecorder()

	ProjectConversationsHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	payload := Conversation{}
	mustDecodeJSON(t, recorder.Body.Bytes(), &payload)
	if payload.ProjectID != projectID {
		t.Fatalf("expected project_id %s, got %#v", projectID, payload)
	}
	if payload.ModelConfigID != modelConfigID {
		t.Fatalf("expected model_config_id %s, got %#v", modelConfigID, payload)
	}
}

func TestConversationInputSubmitHandlerUsesCQRSCommandServiceWhenFeatureEnabled(t *testing.T) {
	t.Setenv("FEATURE_CQRS", "true")
	state, conversationID := seedConversationMessageValidationState(t)
	now := time.Now().UTC().Format(time.RFC3339)
	state.sessionCommands = &sessionCommandApplicationStub{
		submitResult: appcommands.SubmitMessageResult{RunID: "exec_submit_cqrs"},
		onSubmit: func(cmd appcommands.SubmitMessageCommand) {
			state.executions["exec_submit_cqrs"] = Execution{
				ID:             "exec_submit_cqrs",
				WorkspaceID:    localWorkspaceID,
				ConversationID: cmd.SessionID,
				MessageID:      "msg_submit_cqrs",
				State:          RunStateQueued,
				Mode:           PermissionModeDefault,
				ModelID:        "gpt-5.3",
				ModeSnapshot:   PermissionModeDefault,
				ModelSnapshot:  ModelSnapshot{ConfigID: "rc_model_allowed", ModelID: "gpt-5.3"},
				QueueIndex:     0,
				TraceID:        "tr_submit_cqrs",
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			state.conversations[cmd.SessionID] = Conversation{
				ID:            cmd.SessionID,
				WorkspaceID:   localWorkspaceID,
				ProjectID:     "proj_validation",
				Name:          "Validation Conversation",
				QueueState:    QueueStateQueued,
				DefaultMode:   PermissionModeDefault,
				ModelConfigID: "rc_model_allowed",
				CreatedAt:     now,
				UpdatedAt:     now,
			}
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+conversationID+"/runs", strings.NewReader(`{"raw_input":"submit via cqrs","mode":"default","model_config_id":"rc_model_allowed","selected_capabilities":["rule:rc_rule_allowed"],"catalog_revision":"rev_cqrs"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("session_id", conversationID)
	recorder := httptest.NewRecorder()

	ConversationInputSubmitHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	stub := state.sessionCommands.(*sessionCommandApplicationStub)
	if !stub.submitCalled {
		t.Fatalf("expected CQRS submit command service to be called")
	}
	if stub.submitCmd.SessionID != conversationID {
		t.Fatalf("expected session_id %s, got %#v", conversationID, stub.submitCmd)
	}
	if stub.submitCmd.RawInput != "submit via cqrs" {
		t.Fatalf("expected raw_input propagated, got %#v", stub.submitCmd)
	}
	if stub.submitCmd.ModelConfigID != "rc_model_allowed" {
		t.Fatalf("expected model_config_id propagated, got %#v", stub.submitCmd)
	}
	if !strings.Contains(recorder.Body.String(), "exec_submit_cqrs") {
		t.Fatalf("expected response to contain cqrs run id, got %s", recorder.Body.String())
	}
}

func TestConversationInputSubmitHandlerEnqueuesRunViaCQRSCommandAdapter(t *testing.T) {
	t.Setenv("FEATURE_CQRS", "true")
	state, conversationID := seedConversationMessageValidationState(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+conversationID+"/runs", strings.NewReader(`{"raw_input":"submit through adapter","model_config_id":"rc_model_allowed"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("session_id", conversationID)
	recorder := httptest.NewRecorder()

	ConversationInputSubmitHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	payload := ComposerSubmitResponse{}
	mustDecodeJSON(t, recorder.Body.Bytes(), &payload)
	if payload.Run == nil {
		t.Fatalf("expected run payload, got %#v", payload)
	}
	if payload.Run.ConversationID != conversationID {
		t.Fatalf("expected run conversation_id %s, got %#v", conversationID, payload.Run)
	}
	if len(state.executions) != 1 {
		t.Fatalf("expected one execution to be created, got %d", len(state.executions))
	}
}

func TestRunControlHandlerUsesCQRSCommandServiceWhenFeatureEnabled(t *testing.T) {
	t.Setenv("FEATURE_CQRS", "true")
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	conversationID := "conv_control_cqrs"
	runID := "exec_control_cqrs"
	activeRunID := runID
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_control_cqrs",
		Name:              "Control CQRS",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_allowed",
		ActiveExecutionID: &activeRunID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[runID] = Execution{
		ID:             runID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_control_cqrs",
		State:          RunStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_control_cqrs",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.conversationExecutionOrder[conversationID] = []string{runID}
	state.sessionCommands = &sessionCommandApplicationStub{
		controlResult: appcommands.ControlRunResult{OK: true},
		onControl: func(cmd appcommands.ControlRunCommand) {
			execution := state.executions[cmd.RunID]
			execution.State = RunStateCancelled
			execution.UpdatedAt = now
			state.executions[cmd.RunID] = execution
			conversation := state.conversations[execution.ConversationID]
			conversation.QueueState = QueueStateIdle
			conversation.ActiveExecutionID = nil
			conversation.UpdatedAt = now
			state.conversations[execution.ConversationID] = conversation
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/runs/"+runID+"/control", strings.NewReader(`{"action":"stop"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("run_id", runID)
	recorder := httptest.NewRecorder()

	RunControlHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	stub := state.sessionCommands.(*sessionCommandApplicationStub)
	if !stub.controlCalled {
		t.Fatalf("expected CQRS control command service to be called")
	}
	if stub.controlCmd.RunID != runID || stub.controlCmd.Action != "stop" {
		t.Fatalf("expected run control payload propagated, got %#v", stub.controlCmd)
	}
	if !strings.Contains(recorder.Body.String(), "\"state\":\"cancelled\"") {
		t.Fatalf("expected response to contain cancelled state, got %s", recorder.Body.String())
	}
}

func TestRunControlHandlerStopsRunViaCQRSCommandAdapter(t *testing.T) {
	t.Setenv("FEATURE_CQRS", "true")
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	conversationID := "conv_control_cqrs_real"
	runID := "exec_control_cqrs_real"
	activeRunID := runID

	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_control_cqrs_real",
		Name:              "Control CQRS Real",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_allowed",
		ActiveExecutionID: &activeRunID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[runID] = Execution{
		ID:             runID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_control_cqrs_real",
		State:          RunStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_control_cqrs_real",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.conversationExecutionOrder[conversationID] = []string{runID}

	req := httptest.NewRequest(http.MethodPost, "/v1/runs/"+runID+"/control", strings.NewReader(`{"action":"stop"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("run_id", runID)
	recorder := httptest.NewRecorder()

	RunControlHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	state.mu.RLock()
	execution := state.executions[runID]
	state.mu.RUnlock()
	if execution.State != RunStateCancelled {
		t.Fatalf("expected execution cancelled, got %#v", execution)
	}
}
