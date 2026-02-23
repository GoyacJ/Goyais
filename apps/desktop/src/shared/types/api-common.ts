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
export type InspectorTabKey = "diff" | "run" | "files" | "risk";
export type MessageRole = "user" | "assistant" | "system";
export type DiffChangeType = "added" | "modified" | "deleted";

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

export type PaginationQuery = {
  cursor?: string;
  limit?: number;
};

export type ListEnvelope<T> = {
  items: T[];
  next_cursor: string | null;
};

export type StandardError = {
  code: string;
  message: string;
  details: Record<string, unknown>;
  trace_id: string;
};

export type MenuVisibility = Record<MenuKey, PermissionVisibility>;

export type DiffCapability = {
  can_commit: boolean;
  can_discard: boolean;
  can_export_patch: boolean;
  reason?: string;
};
