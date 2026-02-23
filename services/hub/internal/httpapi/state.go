package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"
)

const localWorkspaceID = "ws_local"

type AppState struct {
	mu sync.RWMutex

	workspaces map[string]Workspace
	sessions   map[string]Session

	projects                   map[string]Project
	projectConfigs             map[string]ProjectConfig
	conversations              map[string]Conversation
	conversationMessages       map[string][]ConversationMessage
	conversationSnapshots      map[string][]ConversationSnapshot
	conversationExecutionOrder map[string][]string
	executions                 map[string]Execution

	resources     map[string]Resource
	shareRequests map[string]ShareRequest

	adminUsers map[string]AdminUser
	adminRoles map[Role]AdminRole
	adminAudit []AdminAuditEvent

	modelCatalog map[string][]ModelCatalogItem
}

func NewAppState() *AppState {
	state := &AppState{
		workspaces:                 map[string]Workspace{},
		sessions:                   map[string]Session{},
		projects:                   map[string]Project{},
		projectConfigs:             map[string]ProjectConfig{},
		conversations:              map[string]Conversation{},
		conversationMessages:       map[string][]ConversationMessage{},
		conversationSnapshots:      map[string][]ConversationSnapshot{},
		conversationExecutionOrder: map[string][]string{},
		executions:                 map[string]Execution{},
		resources:                  map[string]Resource{},
		shareRequests:              map[string]ShareRequest{},
		adminUsers:                 map[string]AdminUser{},
		adminRoles:                 map[Role]AdminRole{},
		adminAudit:                 []AdminAuditEvent{},
		modelCatalog:               map[string][]ModelCatalogItem{},
	}

	now := time.Now().UTC().Format(time.RFC3339)
	state.workspaces[localWorkspaceID] = Workspace{
		ID:             localWorkspaceID,
		Name:           "Local",
		Mode:           WorkspaceModeLocal,
		HubURL:         nil,
		IsDefaultLocal: true,
		CreatedAt:      now,
		LoginDisabled:  true,
		AuthMode:       AuthModeDisabled,
	}
	state.modelCatalog[localWorkspaceID] = defaultCatalog(localWorkspaceID, now)
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

	return state
}

func (s *AppState) ListWorkspaces() []Workspace {
	s.mu.RLock()
	defer s.mu.RUnlock()

	local := make([]Workspace, 0, 1)
	remote := make([]Workspace, 0)
	for _, workspace := range s.workspaces {
		if workspace.ID == localWorkspaceID {
			local = append(local, workspace)
			continue
		}
		remote = append(remote, workspace)
	}

	sort.Slice(remote, func(i, j int) bool {
		return strings.ToLower(remote[i].Name) < strings.ToLower(remote[j].Name)
	})

	items := make([]Workspace, 0, len(local)+len(remote))
	items = append(items, local...)
	items = append(items, remote...)
	return items
}

func (s *AppState) GetWorkspace(id string) (Workspace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	workspace, ok := s.workspaces[id]
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

	s.mu.Lock()
	s.workspaces[workspace.ID] = workspace
	s.modelCatalog[workspace.ID] = defaultCatalog(workspace.ID, workspace.CreatedAt)
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

	s.AppendAudit(AdminAuditEvent{
		Actor:    "system",
		Action:   "workspace.create_remote",
		Resource: workspace.ID,
		Result:   "success",
	})
	return workspace
}

func (s *AppState) HasAnyRemoteWorkspace() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, workspace := range s.workspaces {
		if workspace.Mode == WorkspaceModeRemote {
			return true
		}
	}
	return false
}

func (s *AppState) SetSession(session Session) {
	s.mu.Lock()
	s.sessions[session.Token] = session
	s.mu.Unlock()
}

func (s *AppState) GetSession(token string) (Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[token]
	return session, ok
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

func defaultCatalog(workspaceID string, now string) []ModelCatalogItem {
	return []ModelCatalogItem{
		{WorkspaceID: workspaceID, Vendor: "OpenAI", ModelID: "gpt-4.1", Enabled: true, Status: "active", SyncedAt: now},
		{WorkspaceID: workspaceID, Vendor: "Google", ModelID: "gemini-2.0-flash", Enabled: true, Status: "active", SyncedAt: now},
		{WorkspaceID: workspaceID, Vendor: "Qwen", ModelID: "qwen-max", Enabled: true, Status: "active", SyncedAt: now},
		{WorkspaceID: workspaceID, Vendor: "Doubao", ModelID: "doubao-pro", Enabled: true, Status: "preview", SyncedAt: now},
		{WorkspaceID: workspaceID, Vendor: "Zhipu", ModelID: "glm-4.6", Enabled: true, Status: "active", SyncedAt: now},
		{WorkspaceID: workspaceID, Vendor: "MiniMax", ModelID: "abab6.5-chat", Enabled: false, Status: "deprecated", SyncedAt: now},
		{WorkspaceID: workspaceID, Vendor: "Local", ModelID: "llama3.1:8b", Enabled: true, Status: "active", SyncedAt: now},
	}
}

func defaultRoles() map[Role]AdminRole {
	return map[Role]AdminRole{
		RoleViewer: {
			Key:         RoleViewer,
			Name:        "Viewer",
			Permissions: []string{"read"},
			Enabled:     true,
		},
		RoleDeveloper: {
			Key:         RoleDeveloper,
			Name:        "Developer",
			Permissions: []string{"read", "write", "execute"},
			Enabled:     true,
		},
		RoleApprover: {
			Key:         RoleApprover,
			Name:        "Approver",
			Permissions: []string{"read", "approve"},
			Enabled:     true,
		},
		RoleAdmin: {
			Key:         RoleAdmin,
			Name:        "Admin",
			Permissions: []string{"*"},
			Enabled:     true,
		},
	}
}

func randomHex(bytes int) string {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return GenerateTraceID()
	}
	return hex.EncodeToString(buf)
}
