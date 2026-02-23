export type WorkspaceMode = "local" | "remote";
export type AuthMode = "disabled" | "password_or_token" | "token_only";
export type Role = "viewer" | "developer" | "approver" | "admin";
export type QueueState = "idle" | "running" | "queued";
export type ConversationMode = "agent" | "plan";
export type ExecutionState = "queued" | "pending" | "executing" | "confirming" | "completed" | "failed" | "cancelled";
export type PermissionVisibility = "hidden" | "disabled" | "readonly" | "enabled";
export type ConnectionStatus = "connected" | "reconnecting" | "disconnected";
export type ResourceType = "model" | "rule" | "skill" | "mcp";
export type ResourceScope = "private" | "shared";
export type ShareStatus = "pending" | "approved" | "denied" | "revoked";
export type ModelVendorName = "OpenAI" | "Google" | "Qwen" | "Doubao" | "Zhipu" | "MiniMax" | "Local";
export type MenuKey =
  | "main"
  | "remote_account"
  | "remote_members_roles"
  | "remote_permissions_audit"
  | "workspace_project_config"
  | "workspace_agent"
  | "workspace_model"
  | "workspace_rules"
  | "workspace_skills"
  | "workspace_mcp"
  | "settings_theme"
  | "settings_i18n"
  | "settings_updates_diagnostics"
  | "settings_general";

export type Workspace = {
  id: string;
  name: string;
  mode: WorkspaceMode;
  hub_url: string | null;
  is_default_local: boolean;
  created_at: string;
  login_disabled: boolean;
  auth_mode: AuthMode;
};

export type WorkspaceConnection = {
  workspace_id: string;
  hub_url: string;
  username: string;
  connection_status: ConnectionStatus;
  connected_at: string;
  access_token?: string;
};

export type WorkspaceConnectionResult = {
  workspace: Workspace;
  connection: WorkspaceConnection;
  access_token?: string;
};

export type ListEnvelope<T> = {
  items: T[];
  next_cursor: string | null;
};

export type CreateWorkspaceRequest = {
  name?: string;
  hub_url: string;
  username: string;
  password: string;
  login_disabled?: boolean;
  auth_mode?: Exclude<AuthMode, "disabled">;
};

export type LoginRequest = {
  workspace_id: string;
  username?: string;
  password?: string;
  token?: string;
};

export type LoginResponse = {
  access_token: string;
  token_type: "bearer";
  expires_in?: number;
};

export type Capabilities = {
  admin_console: boolean;
  resource_write: boolean;
  execution_control: boolean;
};

export type Me = {
  user_id: string;
  display_name: string;
  workspace_id: string;
  role: Role;
  capabilities: Capabilities;
};

export type StandardError = {
  code: string;
  message: string;
  details: Record<string, unknown>;
  trace_id: string;
};

export type Project = {
  id: string;
  workspace_id: string;
  name: string;
  repo_path: string;
  is_git: boolean;
  default_model_id?: string;
  default_mode?: ConversationMode;
  created_at: string;
  updated_at: string;
};

export type ProjectConfig = {
  project_id: string;
  model_id: string | null;
  rule_ids: string[];
  skill_ids: string[];
  mcp_ids: string[];
  updated_at: string;
};

export type Conversation = {
  id: string;
  workspace_id: string;
  project_id: string;
  name: string;
  queue_state: QueueState;
  default_mode: ConversationMode;
  model_id: string;
  active_execution_id: string | null;
  created_at: string;
  updated_at: string;
};

export type InspectorTabKey = "diff" | "run" | "files" | "risk";

export type MessageRole = "user" | "assistant" | "system";

export type ConversationMessage = {
  id: string;
  conversation_id: string;
  role: MessageRole;
  content: string;
  created_at: string;
  queue_index?: number;
  can_rollback?: boolean;
};

export type ConversationSnapshot = {
  id: string;
  conversation_id: string;
  rollback_point_message_id: string;
  queue_state: QueueState;
  worktree_ref: string | null;
  inspector_state: {
    tab: InspectorTabKey;
  };
  messages: ConversationMessage[];
  execution_ids: string[];
  created_at: string;
};

export type Execution = {
  id: string;
  workspace_id: string;
  conversation_id: string;
  message_id: string;
  state: ExecutionState;
  mode: ConversationMode;
  model_id: string;
  queue_index: number;
  trace_id: string;
  created_at: string;
  updated_at: string;
};

export type ExecutionEventType =
  | "message_received"
  | "execution_started"
  | "thinking_delta"
  | "tool_call"
  | "tool_result"
  | "confirmation_required"
  | "confirmation_resolved"
  | "diff_generated"
  | "execution_stopped"
  | "execution_done"
  | "execution_error";

export type ExecutionEvent = {
  event_id: string;
  execution_id: string;
  conversation_id: string;
  trace_id: string;
  sequence: number;
  queue_index: number;
  type: ExecutionEventType;
  timestamp: string;
  payload: Record<string, unknown>;
};

export type DiffChangeType = "added" | "modified" | "deleted";

export type DiffItem = {
  id: string;
  path: string;
  change_type: DiffChangeType;
  summary: string;
};

export type ExecutionCreateRequest = {
  content: string;
  mode: ConversationMode;
  model_id: string;
};

export type ExecutionCreateResponse = {
  execution: Execution;
};

export type Resource = {
  id: string;
  workspace_id: string;
  type: ResourceType;
  name: string;
  source: "workspace_native" | "local_import";
  scope: ResourceScope;
  share_status: ShareStatus;
  owner_user_id: string;
  enabled: boolean;
  description?: string;
  created_at: string;
  updated_at: string;
};

export type ModelVendor = {
  workspace_id: string;
  name: ModelVendorName;
  enabled: boolean;
  updated_at: string;
};

export type ModelCatalogItem = {
  workspace_id: string;
  vendor: ModelVendorName;
  model_id: string;
  enabled: boolean;
  status: "active" | "deprecated" | "preview";
  synced_at: string;
};

export type ResourceImportRequest = {
  resource_type: ResourceType;
  source_id: string;
  target_workspace_id: string;
};

export type ShareRequest = {
  id: string;
  workspace_id: string;
  resource_id: string;
  status: ShareStatus;
  requester_user_id: string;
  approver_user_id?: string;
  created_at: string;
  updated_at: string;
};

export type AdminUser = {
  id: string;
  workspace_id: string;
  username: string;
  display_name: string;
  role: Role;
  enabled: boolean;
  created_at: string;
};

export type AdminRole = {
  key: Role;
  name: string;
  permissions: string[];
  enabled: boolean;
};

export type AdminAuditEvent = {
  id: string;
  actor: string;
  action: string;
  resource: string;
  result: "success" | "denied" | "failed";
  trace_id: string;
  timestamp: string;
};

export type MenuVisibility = Record<MenuKey, PermissionVisibility>;

export type DiffCapability = {
  can_commit: boolean;
  can_discard: boolean;
  can_export_patch: boolean;
  reason?: string;
};
