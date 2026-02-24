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

type PermissionVisibility string

const (
	PermissionVisibilityHidden   PermissionVisibility = "hidden"
	PermissionVisibilityDisabled PermissionVisibility = "disabled"
	PermissionVisibilityReadonly PermissionVisibility = "readonly"
	PermissionVisibilityEnabled  PermissionVisibility = "enabled"
)

type ABACEffect string

const (
	ABACEffectAllow ABACEffect = "allow"
	ABACEffectDeny  ABACEffect = "deny"
)

type QueueState string

const (
	QueueStateIdle    QueueState = "idle"
	QueueStateRunning QueueState = "running"
	QueueStateQueued  QueueState = "queued"
)

type ConversationMode string

const (
	ConversationModeAgent ConversationMode = "agent"
	ConversationModePlan  ConversationMode = "plan"
)

type ExecutionState string

const (
	ExecutionStateQueued    ExecutionState = "queued"
	ExecutionStatePending   ExecutionState = "pending"
	ExecutionStateExecuting ExecutionState = "executing"
	ExecutionStateCompleted ExecutionState = "completed"
	ExecutionStateFailed    ExecutionState = "failed"
	ExecutionStateCancelled ExecutionState = "cancelled"
)

type ResourceType string

const (
	ResourceTypeModel ResourceType = "model"
	ResourceTypeRule  ResourceType = "rule"
	ResourceTypeSkill ResourceType = "skill"
	ResourceTypeMCP   ResourceType = "mcp"
)

type ShareStatus string

const (
	ShareStatusPending  ShareStatus = "pending"
	ShareStatusApproved ShareStatus = "approved"
	ShareStatusDenied   ShareStatus = "denied"
	ShareStatusRevoked  ShareStatus = "revoked"
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

type WorkspaceConnection struct {
	WorkspaceID      string `json:"workspace_id"`
	HubURL           string `json:"hub_url"`
	Username         string `json:"username"`
	ConnectionStatus string `json:"connection_status"`
	ConnectedAt      string `json:"connected_at"`
	AccessToken      string `json:"access_token,omitempty"`
}

type WorkspaceConnectionResult struct {
	Workspace   Workspace           `json:"workspace"`
	Connection  WorkspaceConnection `json:"connection"`
	AccessToken string              `json:"access_token,omitempty"`
}

type ConversationStatus string

const (
	ConversationStatusRunning ConversationStatus = "running"
	ConversationStatusQueued  ConversationStatus = "queued"
	ConversationStatusStopped ConversationStatus = "stopped"
	ConversationStatusDone    ConversationStatus = "done"
	ConversationStatusError   ConversationStatus = "error"
)

type WorkspaceAgentConfigTraceDetailLevel string

const (
	WorkspaceAgentTraceDetailLevelBasic   WorkspaceAgentConfigTraceDetailLevel = "basic"
	WorkspaceAgentTraceDetailLevelVerbose WorkspaceAgentConfigTraceDetailLevel = "verbose"
)

type WorkspaceStatusResponse struct {
	WorkspaceID        string             `json:"workspace_id"`
	ConversationID     string             `json:"conversation_id,omitempty"`
	ConversationStatus ConversationStatus `json:"conversation_status"`
	HubURL             string             `json:"hub_url"`
	ConnectionStatus   string             `json:"connection_status"`
	UserDisplayName    string             `json:"user_display_name"`
	UpdatedAt          string             `json:"updated_at"`
}

type WorkspaceAgentExecutionConfig struct {
	MaxModelTurns int `json:"max_model_turns"`
}

type WorkspaceAgentDisplayConfig struct {
	ShowProcessTrace bool                                 `json:"show_process_trace"`
	TraceDetailLevel WorkspaceAgentConfigTraceDetailLevel `json:"trace_detail_level"`
}

type WorkspaceAgentConfig struct {
	WorkspaceID string                        `json:"workspace_id"`
	Execution   WorkspaceAgentExecutionConfig `json:"execution"`
	Display     WorkspaceAgentDisplayConfig   `json:"display"`
	UpdatedAt   string                        `json:"updated_at"`
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
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    *int   `json:"expires_in,omitempty"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	AccessToken string `json:"access_token,omitempty"`
}

type RemoteConnectRequest struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	Name        string `json:"name,omitempty"`
	HubURL      string `json:"hub_url,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	Token       string `json:"token,omitempty"`
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
	Token            string    `json:"token"`
	RefreshToken     string    `json:"refresh_token"`
	WorkspaceID      string    `json:"workspace_id"`
	Role             Role      `json:"role"`
	UserID           string    `json:"user_id"`
	DisplayName      string    `json:"display_name"`
	ExpiresAt        time.Time `json:"expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	Revoked          bool      `json:"revoked"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Project struct {
	ID              string           `json:"id"`
	WorkspaceID     string           `json:"workspace_id"`
	Name            string           `json:"name"`
	RepoPath        string           `json:"repo_path"`
	IsGit           bool             `json:"is_git"`
	DefaultModelID  string           `json:"default_model_id,omitempty"`
	DefaultMode     ConversationMode `json:"default_mode,omitempty"`
	CurrentRevision int64            `json:"current_revision"`
	CreatedAt       string           `json:"created_at"`
	UpdatedAt       string           `json:"updated_at"`
}

type ProjectConfig struct {
	ProjectID      string   `json:"project_id"`
	ModelIDs       []string `json:"model_ids"`
	DefaultModelID *string  `json:"default_model_id,omitempty"`
	RuleIDs        []string `json:"rule_ids"`
	SkillIDs       []string `json:"skill_ids"`
	MCPIDs         []string `json:"mcp_ids"`
	UpdatedAt      string   `json:"updated_at"`
}

type CreateProjectRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	RepoPath    string `json:"repo_path"`
	IsGit       bool   `json:"is_git"`
}

type ImportProjectRequest struct {
	WorkspaceID   string `json:"workspace_id"`
	DirectoryPath string `json:"directory_path"`
}

type CreateConversationRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
}

type RenameConversationRequest struct {
	Name string `json:"name"`
}

type UpdateConversationRequest struct {
	Name    *string           `json:"name,omitempty"`
	Mode    *ConversationMode `json:"mode,omitempty"`
	ModelID *string           `json:"model_id,omitempty"`
}

type Conversation struct {
	ID                string           `json:"id"`
	WorkspaceID       string           `json:"workspace_id"`
	ProjectID         string           `json:"project_id"`
	Name              string           `json:"name"`
	QueueState        QueueState       `json:"queue_state"`
	DefaultMode       ConversationMode `json:"default_mode"`
	ModelID           string           `json:"model_id"`
	BaseRevision      int64            `json:"base_revision"`
	ActiveExecutionID *string          `json:"active_execution_id"`
	CreatedAt         string           `json:"created_at"`
	UpdatedAt         string           `json:"updated_at"`
}

type ConversationDetailResponse struct {
	Conversation Conversation           `json:"conversation"`
	Messages     []ConversationMessage  `json:"messages"`
	Executions   []Execution            `json:"executions"`
	Snapshots    []ConversationSnapshot `json:"snapshots"`
}

type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
)

type ConversationMessage struct {
	ID             string      `json:"id"`
	ConversationID string      `json:"conversation_id"`
	Role           MessageRole `json:"role"`
	Content        string      `json:"content"`
	CreatedAt      string      `json:"created_at"`
	QueueIndex     *int        `json:"queue_index,omitempty"`
	CanRollback    *bool       `json:"can_rollback,omitempty"`
}

type ConversationSnapshot struct {
	ID                     string                `json:"id"`
	ConversationID         string                `json:"conversation_id"`
	RollbackPointMessageID string                `json:"rollback_point_message_id"`
	QueueState             QueueState            `json:"queue_state"`
	WorktreeRef            *string               `json:"worktree_ref"`
	InspectorState         ConversationInspector `json:"inspector_state"`
	Messages               []ConversationMessage `json:"messages"`
	ExecutionIDs           []string              `json:"execution_ids"`
	CreatedAt              string                `json:"created_at"`
}

type ConversationInspector struct {
	Tab string `json:"tab"`
}

type Execution struct {
	ID                      string                        `json:"id"`
	WorkspaceID             string                        `json:"workspace_id"`
	ConversationID          string                        `json:"conversation_id"`
	MessageID               string                        `json:"message_id"`
	State                   ExecutionState                `json:"state"`
	Mode                    ConversationMode              `json:"mode"`
	ModelID                 string                        `json:"model_id"`
	ModeSnapshot            ConversationMode              `json:"mode_snapshot"`
	ModelSnapshot           ModelSnapshot                 `json:"model_snapshot"`
	AgentConfigSnapshot     *ExecutionAgentConfigSnapshot `json:"agent_config_snapshot,omitempty"`
	TokensIn                int                           `json:"tokens_in"`
	TokensOut               int                           `json:"tokens_out"`
	ProjectRevisionSnapshot int64                         `json:"project_revision_snapshot"`
	QueueIndex              int                           `json:"queue_index"`
	TraceID                 string                        `json:"trace_id"`
	CreatedAt               string                        `json:"created_at"`
	UpdatedAt               string                        `json:"updated_at"`
}

type ExecutionAgentConfigSnapshot struct {
	MaxModelTurns    int                                  `json:"max_model_turns"`
	ShowProcessTrace bool                                 `json:"show_process_trace"`
	TraceDetailLevel WorkspaceAgentConfigTraceDetailLevel `json:"trace_detail_level"`
}

type ModelSnapshot struct {
	ConfigID  string         `json:"config_id,omitempty"`
	Vendor    string         `json:"vendor,omitempty"`
	ModelID   string         `json:"model_id"`
	BaseURL   string         `json:"base_url,omitempty"`
	TimeoutMS int            `json:"timeout_ms,omitempty"`
	Params    map[string]any `json:"params,omitempty"`
}

type ExecutionCreateRequest struct {
	Content string           `json:"content"`
	Mode    ConversationMode `json:"mode"`
	ModelID string           `json:"model_id"`
}

type ExecutionCreateResponse struct {
	Execution  Execution  `json:"execution"`
	QueueState QueueState `json:"queue_state"`
	QueueIndex int        `json:"queue_index"`
}

type RollbackRequest struct {
	MessageID string `json:"message_id"`
}

type DiffItem struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	ChangeType string `json:"change_type"`
	Summary    string `json:"summary"`
}

type ProjectFileEntry struct {
	Path  string `json:"path"`
	Type  string `json:"type"`
	Size  int64  `json:"size,omitempty"`
	MTime string `json:"mtime,omitempty"`
}

type ProjectFileContentResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type ExecutionEventType string

const (
	ExecutionEventTypeMessageReceived  ExecutionEventType = "message_received"
	ExecutionEventTypeExecutionStarted ExecutionEventType = "execution_started"
	ExecutionEventTypeThinkingDelta    ExecutionEventType = "thinking_delta"
	ExecutionEventTypeToolCall         ExecutionEventType = "tool_call"
	ExecutionEventTypeToolResult       ExecutionEventType = "tool_result"
	ExecutionEventTypeDiffGenerated    ExecutionEventType = "diff_generated"
	ExecutionEventTypeExecutionStopped ExecutionEventType = "execution_stopped"
	ExecutionEventTypeExecutionDone    ExecutionEventType = "execution_done"
	ExecutionEventTypeExecutionError   ExecutionEventType = "execution_error"
)

type ExecutionEvent struct {
	EventID        string             `json:"event_id"`
	ExecutionID    string             `json:"execution_id"`
	ConversationID string             `json:"conversation_id"`
	TraceID        string             `json:"trace_id"`
	Sequence       int                `json:"sequence"`
	QueueIndex     int                `json:"queue_index"`
	Type           ExecutionEventType `json:"type"`
	Timestamp      string             `json:"timestamp"`
	Payload        map[string]any     `json:"payload"`
}

type ExecutionEventBatchRequest struct {
	Events []ExecutionEvent `json:"events"`
}

type ExecutionControlCommandType string

const (
	ExecutionControlCommandTypeStop ExecutionControlCommandType = "stop"
)

type ExecutionControlCommand struct {
	ID          string                      `json:"id"`
	ExecutionID string                      `json:"execution_id"`
	Type        ExecutionControlCommandType `json:"type"`
	Payload     map[string]any              `json:"payload"`
	Seq         int                         `json:"seq"`
	CreatedAt   string                      `json:"created_at"`
}

type ExecutionControlPollResponse struct {
	Commands []ExecutionControlCommand `json:"commands"`
	LastSeq  int                       `json:"last_seq"`
}

type WorkerRegistration struct {
	WorkerID      string         `json:"worker_id"`
	Capabilities  map[string]any `json:"capabilities"`
	Status        string         `json:"status"`
	LastHeartbeat string         `json:"last_heartbeat"`
}

type WorkerRegisterRequest struct {
	WorkerID     string         `json:"worker_id"`
	Capabilities map[string]any `json:"capabilities"`
}

type WorkerHeartbeatRequest struct {
	Status string `json:"status"`
}

type ExecutionLease struct {
	ExecutionID    string `json:"execution_id"`
	WorkerID       string `json:"worker_id"`
	LeaseVersion   int    `json:"lease_version"`
	LeaseExpiresAt string `json:"lease_expires_at"`
	RunAttempt     int    `json:"run_attempt"`
}

type ExecutionClaimRequest struct {
	WorkerID     string `json:"worker_id"`
	LeaseSeconds int    `json:"lease_seconds,omitempty"`
}

type ExecutionClaimEnvelope struct {
	Execution    Execution      `json:"execution"`
	Lease        ExecutionLease `json:"lease"`
	Content      string         `json:"content"`
	ProjectName  string         `json:"project_name,omitempty"`
	ProjectPath  string         `json:"project_path,omitempty"`
	ProjectIsGit bool           `json:"project_is_git"`
}

type ExecutionClaimResponse struct {
	Claimed   bool                    `json:"claimed"`
	Execution *ExecutionClaimEnvelope `json:"execution,omitempty"`
}

type Resource struct {
	ID          string       `json:"id"`
	WorkspaceID string       `json:"workspace_id"`
	Type        ResourceType `json:"type"`
	Name        string       `json:"name"`
	Source      string       `json:"source"`
	Scope       string       `json:"scope"`
	ShareStatus ShareStatus  `json:"share_status"`
	OwnerUserID string       `json:"owner_user_id"`
	Enabled     bool         `json:"enabled"`
	Description string       `json:"description,omitempty"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
}

type ResourceImportRequest struct {
	ResourceType ResourceType `json:"resource_type"`
	SourceID     string       `json:"source_id"`
}

type ShareRequest struct {
	ID              string      `json:"id"`
	WorkspaceID     string      `json:"workspace_id"`
	ResourceID      string      `json:"resource_id"`
	Status          ShareStatus `json:"status"`
	RequesterUserID string      `json:"requester_user_id"`
	ApproverUserID  *string     `json:"approver_user_id,omitempty"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
}

type ModelCatalogItem struct {
	WorkspaceID string `json:"workspace_id"`
	Vendor      string `json:"vendor"`
	ModelID     string `json:"model_id"`
	Enabled     bool   `json:"enabled"`
	Status      string `json:"status"`
	SyncedAt    string `json:"synced_at"`
}

type ModelCatalogSyncRequest struct {
	Vendors []string `json:"vendors"`
}

type AdminUser struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        Role   `json:"role"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at"`
}

type AdminRole struct {
	Key         Role     `json:"key"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	Enabled     bool     `json:"enabled"`
}

type AdminAuditEvent struct {
	ID        string `json:"id"`
	Actor     string `json:"actor"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	Result    string `json:"result"`
	TraceID   string `json:"trace_id"`
	Timestamp string `json:"timestamp"`
}

type PermissionSnapshot struct {
	Role             Role                            `json:"role"`
	Permissions      []string                        `json:"permissions"`
	MenuVisibility   map[string]PermissionVisibility `json:"menu_visibility"`
	ActionVisibility map[string]PermissionVisibility `json:"action_visibility"`
	PolicyVersion    string                          `json:"policy_version"`
	GeneratedAt      string                          `json:"generated_at"`
}

type AdminPermission struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
}

type AdminMenu struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
}

type RoleMenuVisibility struct {
	RoleKey Role                            `json:"role_key"`
	Items   map[string]PermissionVisibility `json:"items"`
}

type ABACPolicy struct {
	ID           string         `json:"id"`
	WorkspaceID  string         `json:"workspace_id"`
	Name         string         `json:"name"`
	Effect       ABACEffect     `json:"effect"`
	Priority     int            `json:"priority"`
	Enabled      bool           `json:"enabled"`
	SubjectExpr  map[string]any `json:"subject_expr"`
	ResourceExpr map[string]any `json:"resource_expr"`
	ActionExpr   map[string]any `json:"action_expr"`
	ContextExpr  map[string]any `json:"context_expr"`
	CreatedAt    string         `json:"created_at,omitempty"`
	UpdatedAt    string         `json:"updated_at,omitempty"`
}
