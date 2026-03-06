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

type PermissionMode string

const (
	PermissionModeDefault           PermissionMode = "default"
	PermissionModeAcceptEdits       PermissionMode = "acceptEdits"
	PermissionModePlan              PermissionMode = "plan"
	PermissionModeDontAsk           PermissionMode = "dontAsk"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// Backward-compatible type alias for existing references while moving to PermissionMode.
type ConversationMode = PermissionMode

type RunState string

const (
	RunStateQueued        RunState = "queued"
	RunStatePending       RunState = "pending"
	RunStateExecuting     RunState = "executing"
	RunStateConfirming    RunState = "confirming"
	RunStateAwaitingInput RunState = "awaiting_input"
	RunStateCompleted     RunState = "completed"
	RunStateFailed        RunState = "failed"
	RunStateCancelled     RunState = "cancelled"
)

type TaskState string

const (
	TaskStateQueued    TaskState = "queued"
	TaskStateBlocked   TaskState = "blocked"
	TaskStateRunning   TaskState = "running"
	TaskStateRetrying  TaskState = "retrying"
	TaskStateCompleted TaskState = "completed"
	TaskStateFailed    TaskState = "failed"
	TaskStateCancelled TaskState = "cancelled"
)

type HookScope string

const (
	HookScopeGlobal  HookScope = "global"
	HookScopeProject HookScope = "project"
	HookScopeLocal   HookScope = "local"
	HookScopePlugin  HookScope = "plugin"
)

type HookEventType string

const (
	HookEventTypeSessionStart       HookEventType = "session_start"
	HookEventTypeSessionEnd         HookEventType = "session_end"
	HookEventTypeUserPromptSubmit   HookEventType = "user_prompt_submit"
	HookEventTypePreToolUse         HookEventType = "pre_tool_use"
	HookEventTypePermissionRequest  HookEventType = "permission_request"
	HookEventTypePostToolUse        HookEventType = "post_tool_use"
	HookEventTypePostToolUseFailure HookEventType = "post_tool_use_failure"
	HookEventTypeSubagentStart      HookEventType = "subagent_start"
	HookEventTypeStop               HookEventType = "stop"
	HookEventTypeSubagentStop       HookEventType = "subagent_stop"
	HookEventTypeTeammateIdle       HookEventType = "teammate_idle"
	HookEventTypeTaskCompleted      HookEventType = "task_completed"
	HookEventTypeNotification       HookEventType = "notification"
	HookEventTypeConfigChange       HookEventType = "config_change"
	HookEventTypeWorktreeCreate     HookEventType = "worktree_create"
	HookEventTypeWorktreeRemove     HookEventType = "worktree_remove"
	HookEventTypePreCompact         HookEventType = "pre_compact"
)

type HookHandlerType string

const (
	HookHandlerTypeCommand HookHandlerType = "command"
	HookHandlerTypeHTTP    HookHandlerType = "http"
	HookHandlerTypePrompt  HookHandlerType = "prompt"
	HookHandlerTypeAgent   HookHandlerType = "agent"
)

type HookDecisionAction string

const (
	HookDecisionActionAllow HookDecisionAction = "allow"
	HookDecisionActionDeny  HookDecisionAction = "deny"
	HookDecisionActionAsk   HookDecisionAction = "ask"
)

type HookDecision struct {
	Action            HookDecisionAction `json:"action"`
	Reason            string             `json:"reason,omitempty"`
	UpdatedInput      map[string]any     `json:"updated_input,omitempty"`
	AdditionalContext map[string]any     `json:"additional_context,omitempty"`
}

type HookPolicy struct {
	ID          string          `json:"id"`
	Scope       HookScope       `json:"scope"`
	Event       HookEventType   `json:"event"`
	HandlerType HookHandlerType `json:"handler_type"`
	ToolName    string          `json:"tool_name,omitempty"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
	ProjectID   string          `json:"project_id,omitempty"`
	SessionID   string          `json:"session_id,omitempty"`
	Enabled     bool            `json:"enabled"`
	Decision    HookDecision    `json:"decision"`
	UpdatedAt   string          `json:"updated_at"`
}

type HookPolicyUpsertRequest struct {
	ID          string          `json:"id"`
	Scope       HookScope       `json:"scope"`
	Event       HookEventType   `json:"event"`
	HandlerType HookHandlerType `json:"handler_type"`
	ToolName    string          `json:"tool_name,omitempty"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
	ProjectID   string          `json:"project_id,omitempty"`
	SessionID   string          `json:"session_id,omitempty"`
	Enabled     *bool           `json:"enabled,omitempty"`
	Decision    HookDecision    `json:"decision"`
}

type HookPolicyListResponse struct {
	Items []HookPolicy `json:"items"`
}

type HookExecutionRecord struct {
	ID        string        `json:"id"`
	RunID     string        `json:"run_id"`
	TaskID    string        `json:"task_id,omitempty"`
	SessionID string        `json:"session_id"`
	Event     HookEventType `json:"event"`
	ToolName  string        `json:"tool_name,omitempty"`
	PolicyID  string        `json:"policy_id,omitempty"`
	Decision  HookDecision  `json:"decision"`
	Timestamp string        `json:"timestamp"`
}

type HookExecutionListResponse struct {
	Items []HookExecutionRecord `json:"items"`
}

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
	WorkspaceID      string             `json:"workspace_id"`
	SessionID        string             `json:"session_id,omitempty"`
	SessionStatus    ConversationStatus `json:"session_status"`
	HubURL           string             `json:"hub_url"`
	ConnectionStatus string             `json:"connection_status"`
	UserDisplayName  string             `json:"user_display_name"`
	UpdatedAt        string             `json:"updated_at"`
}

type WorkspaceAgentExecutionConfig struct {
	MaxModelTurns int `json:"max_model_turns"`
}

type WorkspaceAgentDisplayConfig struct {
	ShowProcessTrace bool                                 `json:"show_process_trace"`
	TraceDetailLevel WorkspaceAgentConfigTraceDetailLevel `json:"trace_detail_level"`
}

type WorkspaceAgentCapabilityBudgets struct {
	PromptBudgetChars     int `json:"prompt_budget_chars"`
	SearchThresholdPercent int `json:"search_threshold_percent"`
}

type WorkspaceAgentMCPSearchConfig struct {
	Enabled    bool `json:"enabled"`
	ResultLimit int `json:"result_limit"`
}

type WorkspaceAgentSubagentDefaults struct {
	MaxTurns     int      `json:"max_turns"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
}

type WorkspaceAgentFeatureFlags struct {
	EnableToolSearch      bool `json:"enable_tool_search"`
	EnableCapabilityGraph bool `json:"enable_capability_graph"`
}

type WorkspaceAgentConfig struct {
	WorkspaceID       string                         `json:"workspace_id"`
	Execution         WorkspaceAgentExecutionConfig  `json:"execution"`
	Display           WorkspaceAgentDisplayConfig    `json:"display"`
	DefaultMode       PermissionMode                 `json:"default_mode"`
	BuiltinTools      []string                       `json:"builtin_tools"`
	CapabilityBudgets WorkspaceAgentCapabilityBudgets `json:"capability_budgets"`
	MCPSearch         WorkspaceAgentMCPSearchConfig  `json:"mcp_search"`
	OutputStyle       string                         `json:"output_style"`
	SubagentDefaults  WorkspaceAgentSubagentDefaults `json:"subagent_defaults"`
	FeatureFlags      WorkspaceAgentFeatureFlags     `json:"feature_flags"`
	UpdatedAt         string                         `json:"updated_at"`
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
	ID                   string           `json:"id"`
	WorkspaceID          string           `json:"workspace_id"`
	Name                 string           `json:"name"`
	RepoPath             string           `json:"repo_path"`
	IsGit                bool             `json:"is_git"`
	DefaultModelConfigID string           `json:"default_model_config_id,omitempty"`
	TokenThreshold       *int             `json:"token_threshold,omitempty"`
	TokensInTotal        int              `json:"tokens_in_total"`
	TokensOutTotal       int              `json:"tokens_out_total"`
	TokensTotal          int              `json:"tokens_total"`
	DefaultMode          ConversationMode `json:"default_mode,omitempty"`
	CurrentRevision      int64            `json:"current_revision"`
	CreatedAt            string           `json:"created_at"`
	UpdatedAt            string           `json:"updated_at"`
}

type ProjectConfig struct {
	ProjectID            string         `json:"project_id"`
	ModelConfigIDs       []string       `json:"model_config_ids"`
	DefaultModelConfigID *string        `json:"default_model_config_id,omitempty"`
	TokenThreshold       *int           `json:"token_threshold,omitempty"`
	ModelTokenThresholds map[string]int `json:"model_token_thresholds"`
	RuleIDs              []string       `json:"rule_ids"`
	SkillIDs             []string       `json:"skill_ids"`
	MCPIDs               []string       `json:"mcp_ids"`
	UpdatedAt            string         `json:"updated_at"`
}

type ModelTokenUsage struct {
	TokensInTotal  int `json:"tokens_in_total"`
	TokensOutTotal int `json:"tokens_out_total"`
	TokensTotal    int `json:"tokens_total"`
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
	Name          *string           `json:"name,omitempty"`
	Mode          *ConversationMode `json:"mode,omitempty"`
	ModelConfigID *string           `json:"model_config_id,omitempty"`
	RuleIDs       []string          `json:"rule_ids,omitempty"`
	SkillIDs      []string          `json:"skill_ids,omitempty"`
	MCPIDs        []string          `json:"mcp_ids,omitempty"`
}

type Conversation struct {
	ID                string           `json:"id"`
	WorkspaceID       string           `json:"workspace_id"`
	ProjectID         string           `json:"project_id"`
	Name              string           `json:"name"`
	QueueState        QueueState       `json:"queue_state"`
	DefaultMode       ConversationMode `json:"default_mode"`
	ModelConfigID     string           `json:"model_config_id"`
	RuleIDs           []string         `json:"rule_ids"`
	SkillIDs          []string         `json:"skill_ids"`
	MCPIDs            []string         `json:"mcp_ids"`
	BaseRevision      int64            `json:"base_revision"`
	ActiveExecutionID *string          `json:"active_run_id"`
	TokensInTotal     int              `json:"tokens_in_total"`
	TokensOutTotal    int              `json:"tokens_out_total"`
	TokensTotal       int              `json:"tokens_total"`
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
	ConversationID string      `json:"session_id"`
	Role           MessageRole `json:"role"`
	Content        string      `json:"content"`
	CreatedAt      string      `json:"created_at"`
	QueueIndex     *int        `json:"queue_index,omitempty"`
	CanRollback    *bool       `json:"can_rollback,omitempty"`
}

type ConversationSnapshot struct {
	ID                     string                `json:"id"`
	ConversationID         string                `json:"session_id"`
	RollbackPointMessageID string                `json:"rollback_point_message_id"`
	QueueState             QueueState            `json:"queue_state"`
	WorktreeRef            *string               `json:"worktree_ref"`
	InspectorState         ConversationInspector `json:"inspector_state"`
	Messages               []ConversationMessage `json:"messages"`
	ExecutionIDs           []string              `json:"run_ids"`
	CreatedAt              string                `json:"created_at"`
}

type ConversationInspector struct {
	Tab string `json:"tab"`
}

type Execution struct {
	ID                      string                        `json:"id"`
	WorkspaceID             string                        `json:"workspace_id"`
	ConversationID          string                        `json:"session_id"`
	MessageID               string                        `json:"message_id"`
	State                   RunState                      `json:"state"`
	Mode                    ConversationMode              `json:"mode"`
	ModelID                 string                        `json:"model_id"`
	ModeSnapshot            ConversationMode              `json:"mode_snapshot"`
	ModelSnapshot           ModelSnapshot                 `json:"model_snapshot"`
	ResourceProfileSnapshot *ExecutionResourceProfile     `json:"resource_profile_snapshot,omitempty"`
	AgentConfigSnapshot     *ExecutionAgentConfigSnapshot `json:"agent_config_snapshot,omitempty"`
	TokensIn                int                           `json:"tokens_in"`
	TokensOut               int                           `json:"tokens_out"`
	ProjectRevisionSnapshot int64                         `json:"project_revision_snapshot"`
	QueueIndex              int                           `json:"queue_index"`
	TraceID                 string                        `json:"trace_id"`
	CreatedAt               string                        `json:"created_at"`
	UpdatedAt               string                        `json:"updated_at"`
}

type TaskArtifact struct {
	TaskID   string         `json:"task_id"`
	Kind     string         `json:"kind"`
	URI      string         `json:"uri,omitempty"`
	Summary  string         `json:"summary,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type TaskNode struct {
	TaskID      string        `json:"task_id"`
	RunID       string        `json:"run_id"`
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	State       TaskState     `json:"state"`
	AgentID     string        `json:"agent_id,omitempty"`
	DependsOn   []string      `json:"depends_on"`
	Children    []string      `json:"children"`
	RetryCount  int           `json:"retry_count"`
	MaxRetries  int           `json:"max_retries"`
	Artifact    *TaskArtifact `json:"artifact,omitempty"`
	LastError   *string       `json:"last_error"`
	CreatedAt   string        `json:"created_at"`
	UpdatedAt   string        `json:"updated_at"`
}

type RunGraphEdge struct {
	FromTaskID string `json:"from_task_id"`
	ToTaskID   string `json:"to_task_id"`
}

type AgentGraph struct {
	RunID          string         `json:"run_id"`
	MaxParallelism int            `json:"max_parallelism"`
	Tasks          []TaskNode     `json:"tasks"`
	Edges          []RunGraphEdge `json:"edges"`
}

type RunTaskListResponse struct {
	Items      []TaskNode `json:"items"`
	NextCursor *string    `json:"next_cursor"`
}

type TaskControlRequest struct {
	Action string `json:"action"`
	Reason string `json:"reason,omitempty"`
}

type TaskControlResponse struct {
	OK            bool   `json:"ok"`
	RunID         string `json:"run_id"`
	TaskID        string `json:"task_id"`
	State         string `json:"state"`
	PreviousState string `json:"previous_state"`
}

type ExecutionAgentConfigSnapshot struct {
	MaxModelTurns     int                                  `json:"max_model_turns"`
	ShowProcessTrace  bool                                 `json:"show_process_trace"`
	TraceDetailLevel  WorkspaceAgentConfigTraceDetailLevel `json:"trace_detail_level"`
	DefaultMode       PermissionMode                       `json:"default_mode"`
	BuiltinTools      []string                             `json:"builtin_tools,omitempty"`
	CapabilityBudgets WorkspaceAgentCapabilityBudgets      `json:"capability_budgets"`
	MCPSearch         WorkspaceAgentMCPSearchConfig        `json:"mcp_search"`
	OutputStyle       string                               `json:"output_style,omitempty"`
	SubagentDefaults  WorkspaceAgentSubagentDefaults       `json:"subagent_defaults"`
	FeatureFlags      WorkspaceAgentFeatureFlags           `json:"feature_flags"`
}

type ExecutionMCPServerSnapshot struct {
	Name      string            `json:"name"`
	Transport string            `json:"transport"`
	Endpoint  string            `json:"endpoint,omitempty"`
	Command   string            `json:"command,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Tools     []string          `json:"tools,omitempty"`
}

type ExecutionCapabilityDescriptorSnapshot struct {
	ID                  string         `json:"id"`
	Kind                string         `json:"kind"`
	Name                string         `json:"name"`
	Description         string         `json:"description"`
	Source              string         `json:"source"`
	Scope               string         `json:"scope"`
	Version             string         `json:"version"`
	InputSchema         map[string]any `json:"input_schema,omitempty"`
	RiskLevel           string         `json:"risk_level"`
	ReadOnly            bool           `json:"read_only"`
	ConcurrencySafe     bool           `json:"concurrency_safe"`
	RequiresPermissions bool           `json:"requires_permissions"`
	VisibilityPolicy    string         `json:"visibility_policy"`
	PromptBudgetCost    int            `json:"prompt_budget_cost"`
}

type ExecutionResourceProfile struct {
	ModelConfigID           string                                `json:"model_config_id,omitempty"`
	ModelID                 string                                `json:"model_id"`
	RuleIDs                 []string                              `json:"rule_ids,omitempty"`
	SkillIDs                []string                              `json:"skill_ids,omitempty"`
	MCPIDs                  []string                              `json:"mcp_ids,omitempty"`
	ProjectFilePaths        []string                              `json:"project_file_paths,omitempty"`
	RulesDSL                string                                `json:"rules_dsl,omitempty"`
	MCPServers              []ExecutionMCPServerSnapshot          `json:"mcp_servers,omitempty"`
	AlwaysLoadedCapabilities []ExecutionCapabilityDescriptorSnapshot `json:"always_loaded_capabilities,omitempty"`
	SearchableCapabilities  []ExecutionCapabilityDescriptorSnapshot `json:"searchable_capabilities,omitempty"`
}

type ModelSnapshot struct {
	ConfigID   string            `json:"config_id,omitempty"`
	Vendor     string            `json:"vendor,omitempty"`
	ModelID    string            `json:"model_id"`
	BaseURL    string            `json:"base_url,omitempty"`
	BaseURLKey string            `json:"base_url_key,omitempty"`
	Runtime    *ModelRuntimeSpec `json:"runtime,omitempty"`
	Params     map[string]any    `json:"params,omitempty"`
}

type ComposerCapabilityKind string

const (
	ComposerCapabilityKindModel ComposerCapabilityKind = "model"
	ComposerCapabilityKindRule  ComposerCapabilityKind = "rule"
	ComposerCapabilityKindSkill ComposerCapabilityKind = "skill"
	ComposerCapabilityKindMCP   ComposerCapabilityKind = "mcp"
	ComposerCapabilityKindFile  ComposerCapabilityKind = "file"
)

type ComposerSubmitRequest struct {
	RawInput             string             `json:"raw_input"`
	Mode                 ConversationMode   `json:"mode"`
	ModelConfigID        string             `json:"model_config_id,omitempty"`
	SelectedCapabilities []string           `json:"selected_capabilities,omitempty"`
	CatalogRevision      string             `json:"catalog_revision,omitempty"`
}

type ComposerCommandCatalogItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}

type ComposerCapabilityCatalogItem struct {
	ID          string                 `json:"id"`
	Kind        ComposerCapabilityKind `json:"kind"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Source      string                 `json:"source,omitempty"`
	Scope       string                 `json:"scope,omitempty"`
}

type ComposerCatalogResponse struct {
	Revision     string                          `json:"revision"`
	Commands     []ComposerCommandCatalogItem    `json:"commands"`
	Capabilities []ComposerCapabilityCatalogItem `json:"capabilities"`
}

type ComposerSuggestRequest struct {
	Draft           string `json:"draft"`
	Cursor          int    `json:"cursor"`
	Limit           int    `json:"limit,omitempty"`
	CatalogRevision string `json:"catalog_revision,omitempty"`
}

type ComposerSuggestion struct {
	Kind         string `json:"kind"`
	Label        string `json:"label"`
	Detail       string `json:"detail,omitempty"`
	InsertText   string `json:"insert_text"`
	ReplaceStart int    `json:"replace_start"`
	ReplaceEnd   int    `json:"replace_end"`
}

type ComposerSuggestResponse struct {
	Revision    string               `json:"revision"`
	Suggestions []ComposerSuggestion `json:"suggestions"`
}

type ComposerCommandResult struct {
	Command string `json:"command"`
	Output  string `json:"output"`
}

type ComposerSubmitResponse struct {
	Kind          string                 `json:"kind"`
	CommandResult *ComposerCommandResult `json:"command_result,omitempty"`
	Run           *Execution             `json:"run,omitempty"`
	QueueState    QueueState             `json:"queue_state,omitempty"`
	QueueIndex    *int                   `json:"queue_index,omitempty"`
}

type RollbackRequest struct {
	MessageID string `json:"message_id"`
}

type DiffItem struct {
	ID           string `json:"id"`
	Path         string `json:"path"`
	ChangeType   string `json:"change_type"`
	Summary      string `json:"summary"`
	AddedLines   *int   `json:"added_lines,omitempty"`
	DeletedLines *int   `json:"deleted_lines,omitempty"`
	BeforeBlob   string `json:"before_blob,omitempty"`
	AfterBlob    string `json:"after_blob,omitempty"`
}

type ExecutionFilesExportResponse struct {
	FileName      string `json:"file_name"`
	ArchiveBase64 string `json:"archive_base64"`
}

type ChangeEntry struct {
	EntryID      string `json:"entry_id"`
	MessageID    string `json:"message_id"`
	ExecutionID  string `json:"run_id"`
	Path         string `json:"path"`
	ChangeType   string `json:"change_type"`
	Summary      string `json:"summary"`
	AddedLines   *int   `json:"added_lines,omitempty"`
	DeletedLines *int   `json:"deleted_lines,omitempty"`
	BeforeBlob   string `json:"before_blob,omitempty"`
	AfterBlob    string `json:"after_blob,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type ChangeSetCapability struct {
	CanCommit  bool   `json:"can_commit"`
	CanDiscard bool   `json:"can_discard"`
	CanExport  bool   `json:"can_export"`
	Reason     string `json:"reason,omitempty"`
}

type CommitSuggestion struct {
	Message string `json:"message"`
}

type CheckpointSummary struct {
	CheckpointID  string `json:"checkpoint_id"`
	Message       string `json:"message"`
	CreatedAt     string `json:"created_at"`
	GitCommitID   string `json:"git_commit_id,omitempty"`
	EntriesDigest string `json:"entries_digest,omitempty"`
}

type ConversationChangeSet struct {
	ChangeSetID             string              `json:"change_set_id"`
	ConversationID          string              `json:"session_id"`
	ProjectKind             string              `json:"project_kind"`
	Entries                 []ChangeEntry       `json:"entries"`
	FileCount               int                 `json:"file_count"`
	AddedLines              int                 `json:"added_lines"`
	DeletedLines            int                 `json:"deleted_lines"`
	Capability              ChangeSetCapability `json:"capability"`
	SuggestedMessage        CommitSuggestion    `json:"suggested_message"`
	LastCommittedCheckpoint *CheckpointSummary  `json:"last_committed_checkpoint,omitempty"`
}

type ChangeSetCommitRequest struct {
	Message             string `json:"message"`
	ExpectedChangeSetID string `json:"expected_change_set_id"`
}

type ChangeSetDiscardRequest struct {
	ExpectedChangeSetID string `json:"expected_change_set_id"`
}

type ChangeSetCommitResponse struct {
	OK         bool              `json:"ok"`
	Checkpoint CheckpointSummary `json:"checkpoint"`
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

type RunEventType string

const (
	RunEventTypeMessageReceived         RunEventType = "message_received"
	RunEventTypeUserPromptSubmit        RunEventType = "user_prompt_submit"
	RunEventTypeExecutionStarted        RunEventType = "execution_started"
	RunEventTypeThinkingDelta           RunEventType = "thinking_delta"
	RunEventTypePreToolUse              RunEventType = "pre_tool_use"
	RunEventTypePermissionRequest       RunEventType = "permission_request"
	RunEventTypeToolCall                RunEventType = "tool_call"
	RunEventTypeToolResult              RunEventType = "tool_result"
	RunEventTypePostToolUse             RunEventType = "post_tool_use"
	RunEventTypePostToolUseFailure      RunEventType = "post_tool_use_failure"
	RunEventTypeDiffGenerated           RunEventType = "diff_generated"
	RunEventTypeChangeSetUpdated        RunEventType = "change_set_updated"
	RunEventTypeChangeSetCommitted      RunEventType = "change_set_committed"
	RunEventTypeChangeSetDiscarded      RunEventType = "change_set_discarded"
	RunEventTypeChangeSetRolledBack     RunEventType = "change_set_rolled_back"
	RunEventTypeExecutionStopped        RunEventType = "execution_stopped"
	RunEventTypeExecutionDone           RunEventType = "execution_done"
	RunEventTypeExecutionError          RunEventType = "execution_error"
	RunEventTypeTaskGraphConfigured     RunEventType = "task_graph_configured"
	RunEventTypeTaskDependenciesUpdated RunEventType = "task_dependencies_updated"
	RunEventTypeTaskRetryPolicyUpdated  RunEventType = "task_retry_policy_updated"
	RunEventTypeTaskArtifactEmitted     RunEventType = "task_artifact_emitted"
	RunEventTypeTaskFailed              RunEventType = "task_failed"
	RunEventTypeTaskStarted             RunEventType = "task_started"
	RunEventTypeTaskCompleted           RunEventType = "task_completed"
	RunEventTypeTaskCancelled           RunEventType = "task_cancelled"
)

type ExecutionEvent struct {
	EventID        string         `json:"event_id"`
	ExecutionID    string         `json:"run_id"`
	ConversationID string         `json:"session_id"`
	TraceID        string         `json:"trace_id"`
	Sequence       int            `json:"sequence"`
	QueueIndex     int            `json:"queue_index"`
	Type           RunEventType   `json:"type"`
	Timestamp      string         `json:"timestamp"`
	Payload        map[string]any `json:"payload"`
}

type ExecutionEventBatchRequest struct {
	Events []ExecutionEvent `json:"events"`
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
