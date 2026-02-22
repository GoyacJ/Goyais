package httpapi

import "time"

type WorkspaceMode string

const (
	WorkspaceModeLocal  WorkspaceMode = "local"
	WorkspaceModeRemote WorkspaceMode = "remote"
)

type AuthMode string

const (
	AuthModeDisabled        AuthMode = "disabled"
	AuthModePasswordOrToken AuthMode = "password_or_token"
	AuthModeTokenOnly       AuthMode = "token_only"
)

type Role string

const (
	RoleViewer    Role = "viewer"
	RoleDeveloper Role = "developer"
	RoleApprover  Role = "approver"
	RoleAdmin     Role = "admin"
)

type Workspace struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Mode           WorkspaceMode `json:"mode"`
	HubURL         *string       `json:"hub_url"`
	IsDefaultLocal bool          `json:"is_default_local"`
	CreatedAt      string        `json:"created_at"`
	LoginDisabled  bool          `json:"login_disabled"`
	AuthMode       AuthMode      `json:"auth_mode"`
}

type CreateWorkspaceRequest struct {
	Name          string   `json:"name"`
	HubURL        string   `json:"hub_url"`
	LoginDisabled *bool    `json:"login_disabled,omitempty"`
	AuthMode      AuthMode `json:"auth_mode,omitempty"`
}

type LoginRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	Token       string `json:"token,omitempty"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   *int   `json:"expires_in,omitempty"`
}

type RemoteConnectRequest struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	Name        string `json:"name,omitempty"`
	HubURL      string `json:"hub_url,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	Token       string `json:"token,omitempty"`
}

type RemoteConnectResponse struct {
	Workspace Workspace     `json:"workspace"`
	Login     LoginResponse `json:"login"`
}

type Capabilities struct {
	AdminConsole     bool `json:"admin_console"`
	ResourceWrite    bool `json:"resource_write"`
	ExecutionControl bool `json:"execution_control"`
}

type Me struct {
	UserID       string       `json:"user_id"`
	DisplayName  string       `json:"display_name"`
	WorkspaceID  string       `json:"workspace_id"`
	Role         Role         `json:"role"`
	Capabilities Capabilities `json:"capabilities"`
}

type Session struct {
	Token       string    `json:"token"`
	WorkspaceID string    `json:"workspace_id"`
	Role        Role      `json:"role"`
	UserID      string    `json:"user_id"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}
