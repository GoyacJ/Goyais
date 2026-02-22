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
	mu         sync.RWMutex
	workspaces map[string]Workspace
	sessions   map[string]Session
}

func NewAppState() *AppState {
	state := &AppState{
		workspaces: map[string]Workspace{},
		sessions:   map[string]Session{},
	}

	state.workspaces[localWorkspaceID] = Workspace{
		ID:             localWorkspaceID,
		Name:           "Local",
		Mode:           WorkspaceModeLocal,
		HubURL:         nil,
		IsDefaultLocal: true,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		LoginDisabled:  true,
		AuthMode:       AuthModeDisabled,
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
	s.mu.Unlock()

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

func randomHex(bytes int) string {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return GenerateTraceID()
	}
	return hex.EncodeToString(buf)
}
