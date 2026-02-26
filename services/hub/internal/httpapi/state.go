package httpapi

import (
	"strings"
	"sync"
	"time"
)

const localWorkspaceID = "ws_local"

type AppState struct {
	mu sync.RWMutex

	authz *authzStore

	workspaces map[string]Workspace
	sessions   map[string]Session

	projects                   map[string]Project
	projectConfigs             map[string]ProjectConfig
	conversations              map[string]Conversation
	conversationMessages       map[string][]ConversationMessage
	conversationSnapshots      map[string][]ConversationSnapshot
	conversationExecutionOrder map[string][]string
	executions                 map[string]Execution
	executionEvents            map[string][]ExecutionEvent
	executionDiffs             map[string][]DiffItem
	conversationEventSeq       map[string]int
	conversationEventSubs      map[string]map[string]chan ExecutionEvent

	resources             map[string]Resource
	resourceConfigs       map[string]ResourceConfig
	resourceTestLogs      []ResourceTestLog
	workspaceCatalogRoots map[string]CatalogRootResponse
	modelCatalogCache     map[string]modelCatalogCacheEntry
	shareRequests         map[string]ShareRequest

	adminUsers map[string]AdminUser
	adminRoles map[Role]AdminRole
	adminAudit []AdminAuditEvent

	orchestrator *ExecutionOrchestrator
}

func NewAppState(store *authzStore) *AppState {
	state := &AppState{
		authz:                      store,
		workspaces:                 map[string]Workspace{},
		sessions:                   map[string]Session{},
		projects:                   map[string]Project{},
		projectConfigs:             map[string]ProjectConfig{},
		conversations:              map[string]Conversation{},
		conversationMessages:       map[string][]ConversationMessage{},
		conversationSnapshots:      map[string][]ConversationSnapshot{},
		conversationExecutionOrder: map[string][]string{},
		executions:                 map[string]Execution{},
		executionEvents:            map[string][]ExecutionEvent{},
		executionDiffs:             map[string][]DiffItem{},
		conversationEventSeq:       map[string]int{},
		conversationEventSubs:      map[string]map[string]chan ExecutionEvent{},
		resources:                  map[string]Resource{},
		resourceConfigs:            map[string]ResourceConfig{},
		resourceTestLogs:           []ResourceTestLog{},
		workspaceCatalogRoots:      map[string]CatalogRootResponse{},
		modelCatalogCache:          map[string]modelCatalogCacheEntry{},
		shareRequests:              map[string]ShareRequest{},
		adminUsers:                 map[string]AdminUser{},
		adminRoles:                 map[Role]AdminRole{},
		adminAudit:                 []AdminAuditEvent{},
	}

	now := time.Now().UTC().Format(time.RFC3339)
	localWorkspace := defaultLocalWorkspace(now)
	state.setWorkspaceCache(localWorkspace)
	if state.authz != nil {
		if _, err := state.authz.upsertWorkspace(localWorkspace); err == nil {
			if persisted, listErr := state.authz.listWorkspaces(); listErr == nil && len(persisted) > 0 {
				state.syncWorkspaceCache(persisted)
			}
		}
		state.hydrateExecutionDomainFromStore()
	}
	state.orchestrator = NewExecutionOrchestrator(state)

	state.adminRoles = defaultRoles()
	state.adminUsers["u_local_admin"] = AdminUser{
		ID:          "u_local_admin",
		WorkspaceID: localWorkspaceID,
		Username:    "local-admin",
		DisplayName: "Local Admin",
		Role:        RoleAdmin,
		Enabled:     true,
		CreatedAt:   now,
	}
	if state.authz != nil {
		for _, workspace := range state.ListWorkspaces() {
			_ = state.authz.ensureWorkspaceSeeds(workspace.ID)
			defaultUser := AdminUser{
				WorkspaceID: workspace.ID,
				Username:    "admin",
				DisplayName: "Remote Admin",
				Role:        RoleAdmin,
				Enabled:     true,
			}
			if workspace.ID == localWorkspaceID {
				defaultUser.Username = "local-admin"
				defaultUser.DisplayName = "Local Admin"
			}
			_, _ = state.authz.upsertUser(defaultUser)
		}
	}
	state.startCatalogWatcher()

	return state
}

func (s *AppState) ListWorkspaces() []Workspace {
	if s.authz != nil {
		items, err := s.authz.listWorkspaces()
		if err == nil {
			s.syncWorkspaceCache(items)
			s.mu.RLock()
			cached := make([]Workspace, 0, len(s.workspaces))
			for _, workspace := range s.workspaces {
				cached = append(cached, workspace)
			}
			s.mu.RUnlock()
			sortWorkspaces(cached)
			return cached
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Workspace, 0, len(s.workspaces))
	for _, workspace := range s.workspaces {
		items = append(items, workspace)
	}
	sortWorkspaces(items)
	return items
}

func (s *AppState) GetWorkspace(id string) (Workspace, bool) {
	workspaceID := strings.TrimSpace(id)
	if workspaceID == "" {
		return Workspace{}, false
	}
	if s.authz != nil {
		workspace, exists, err := s.authz.getWorkspace(workspaceID)
		if err == nil {
			if exists {
				s.setWorkspaceCache(workspace)
			}
			return workspace, exists
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	workspace, ok := s.workspaces[workspaceID]
	return workspace, ok
}

func (s *AppState) CreateRemoteWorkspace(input CreateWorkspaceRequest) Workspace {
	loginDisabled := false
	if input.LoginDisabled != nil {
		loginDisabled = *input.LoginDisabled
	}

	authMode := input.AuthMode
	if authMode == "" {
		authMode = AuthModePasswordOrToken
	}

	hubURL := strings.TrimSpace(input.HubURL)
	workspace := Workspace{
		ID:             "ws_remote_" + randomHex(6),
		Name:           strings.TrimSpace(input.Name),
		Mode:           WorkspaceModeRemote,
		HubURL:         &hubURL,
		IsDefaultLocal: false,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		LoginDisabled:  loginDisabled,
		AuthMode:       authMode,
	}
	if s.authz != nil {
		persisted, err := s.authz.upsertWorkspace(workspace)
		if err == nil {
			workspace = persisted
		}
	}
	s.setWorkspaceCache(workspace)
	s.mu.Lock()
	s.adminUsers["u_"+workspace.ID+"_admin"] = AdminUser{
		ID:          "u_" + workspace.ID + "_admin",
		WorkspaceID: workspace.ID,
		Username:    "admin",
		DisplayName: "Remote Admin",
		Role:        RoleAdmin,
		Enabled:     true,
		CreatedAt:   workspace.CreatedAt,
	}
	s.mu.Unlock()
	if s.authz != nil {
		_ = s.authz.ensureWorkspaceSeeds(workspace.ID)
		_, _ = s.authz.upsertUser(AdminUser{
			WorkspaceID: workspace.ID,
			Username:    "admin",
			DisplayName: "Remote Admin",
			Role:        RoleAdmin,
			Enabled:     true,
		})
		_ = s.authz.appendAudit(workspace.ID, "system", "workspace.create_remote", "workspace", workspace.ID, "success", map[string]any{
			"name":    workspace.Name,
			"hub_url": derefString(workspace.HubURL),
		}, GenerateTraceID())
	} else {
		s.AppendAudit(AdminAuditEvent{
			Actor:    "system",
			Action:   "workspace.create_remote",
			Resource: workspace.ID,
			Result:   "success",
		})
	}
	return workspace
}

func (s *AppState) HasAnyRemoteWorkspace() bool {
	if s.authz != nil {
		hasAny, err := s.authz.hasRemoteWorkspace()
		if err == nil {
			return hasAny
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, workspace := range s.workspaces {
		if workspace.Mode == WorkspaceModeRemote {
			return true
		}
	}
	return false
}

func (s *AppState) SetWorkspaceConnection(connection WorkspaceConnection, actor string, traceID string) {
	if strings.TrimSpace(connection.WorkspaceID) == "" {
		return
	}
	if s.authz != nil {
		_ = s.authz.upsertWorkspaceConnection(connection)
		_ = s.authz.appendAudit(connection.WorkspaceID, firstNonEmpty(strings.TrimSpace(actor), "system"), "workspace.connect", "workspace", connection.WorkspaceID, "success", map[string]any{
			"hub_url":           connection.HubURL,
			"username":          connection.Username,
			"connection_status": connection.ConnectionStatus,
			"connected_at":      connection.ConnectedAt,
		}, strings.TrimSpace(traceID))
		return
	}
	s.AppendAudit(AdminAuditEvent{
		Actor:    firstNonEmpty(strings.TrimSpace(actor), "system"),
		Action:   "workspace.connect",
		Resource: connection.WorkspaceID,
		Result:   "success",
		TraceID:  firstNonEmpty(strings.TrimSpace(traceID), GenerateTraceID()),
	})
}

func (s *AppState) AppendWorkspaceSwitchAudit(workspaceID string, actor string, traceID string) {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	if normalizedWorkspaceID == "" {
		return
	}
	normalizedActor := firstNonEmpty(strings.TrimSpace(actor), "system")
	normalizedTraceID := firstNonEmpty(strings.TrimSpace(traceID), GenerateTraceID())
	if s.authz != nil {
		_ = s.authz.appendAudit(normalizedWorkspaceID, normalizedActor, "workspace.switch_context", "workspace", normalizedWorkspaceID, "success", map[string]any{}, normalizedTraceID)
		return
	}
	s.AppendAudit(AdminAuditEvent{
		Actor:    normalizedActor,
		Action:   "workspace.switch_context",
		Resource: normalizedWorkspaceID,
		Result:   "success",
		TraceID:  normalizedTraceID,
	})
}

func (s *AppState) SetSession(session Session) {
	s.mu.Lock()
	s.sessions[session.Token] = session
	s.mu.Unlock()
	if s.authz != nil {
		_, _ = s.authz.db.Exec(
			`INSERT INTO sessions(access_token, refresh_token, workspace_id, user_id, display_name, role, expires_at, refresh_expires_at, revoked, created_at, updated_at)
			 VALUES(?,?,?,?,?,?,?,?,?,?,?)
			 ON CONFLICT(access_token) DO UPDATE SET refresh_token=excluded.refresh_token, workspace_id=excluded.workspace_id, user_id=excluded.user_id, display_name=excluded.display_name, role=excluded.role, expires_at=excluded.expires_at, refresh_expires_at=excluded.refresh_expires_at, revoked=excluded.revoked, updated_at=excluded.updated_at`,
			session.Token,
			session.RefreshToken,
			session.WorkspaceID,
			session.UserID,
			session.DisplayName,
			string(session.Role),
			session.ExpiresAt.Format(time.RFC3339),
			session.RefreshExpiresAt.Format(time.RFC3339),
			boolToInt(session.Revoked),
			session.CreatedAt.Format(time.RFC3339),
			session.UpdatedAt.Format(time.RFC3339),
		)
	}
}

func (s *AppState) GetSession(token string) (Session, bool) {
	s.mu.RLock()
	session, ok := s.sessions[token]
	s.mu.RUnlock()
	if ok {
		return session, true
	}
	if s.authz != nil {
		persisted, exists, err := s.authz.getSession(token)
		if err != nil || !exists {
			return Session{}, false
		}
		s.mu.Lock()
		s.sessions[token] = persisted
		s.mu.Unlock()
		return persisted, true
	}
	return Session{}, false
}

func (s *AppState) RefreshSession(refreshToken string) (Session, bool) {
	if s.authz == nil {
		return Session{}, false
	}
	session, ok, err := s.authz.refreshSession(refreshToken)
	if err != nil || !ok {
		return Session{}, false
	}
	s.mu.Lock()
	s.sessions[session.Token] = session
	s.mu.Unlock()
	return session, true
}

func (s *AppState) RevokeSession(accessToken string) bool {
	if s.authz != nil {
		if err := s.authz.revokeSession(accessToken); err != nil {
			return false
		}
	}
	s.mu.Lock()
	delete(s.sessions, accessToken)
	s.mu.Unlock()
	return true
}

func (s *AppState) AppendAudit(input AdminAuditEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := input
	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = "audit_" + randomHex(6)
	}
	if strings.TrimSpace(entry.TraceID) == "" {
		entry.TraceID = GenerateTraceID()
	}
	if strings.TrimSpace(entry.Timestamp) == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	s.adminAudit = append([]AdminAuditEvent{entry}, s.adminAudit...)
}
