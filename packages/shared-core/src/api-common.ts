export type WorkspaceMode = "local" | "remote";
export type AuthMode = "disabled" | "password_or_token" | "token_only";
export type Role = "viewer" | "developer" | "approver" | "admin";
export type QueueState = "idle" | "running" | "queued";
export type PermissionMode = "default" | "acceptEdits" | "plan" | "dontAsk" | "bypassPermissions";
// Backward-compatible type name for existing imports.
export type ConversationMode = PermissionMode;
export type ConversationStatus = "running" | "queued" | "stopped" | "done" | "error";
export type ExecutionState =
  | "queued"
  | "pending"
  | "executing"
  | "confirming"
  | "awaiting_input"
  | "completed"
  | "failed"
  | "cancelled";
export type RunState =
  | "queued"
  | "running"
  | "waiting_approval"
  | "waiting_user_input"
  | "completed"
  | "failed"
  | "cancelled";
export type RunControlAction = "stop" | "approve" | "deny" | "resume" | "answer";
export type PermissionVisibility = "hidden" | "disabled" | "readonly" | "enabled";
export type ABACEffect = "allow" | "deny";
export type ConnectionStatus = "connected" | "reconnecting" | "disconnected";
export type ResourceType = "model" | "rule" | "skill" | "mcp";
export type ResourceScope = "private" | "shared";
export type ShareStatus = "pending" | "approved" | "denied" | "revoked";
export type ModelVendorName = "OpenAI" | "DeepSeek" | "Google" | "Qwen" | "Doubao" | "Zhipu" | "MiniMax" | "Local";
export type InspectorTabKey = "diff" | "run" | "trace" | "risk";
export type MessageRole = "user" | "assistant" | "system";
export type DiffChangeType = "added" | "modified" | "deleted";
export type ProjectKind = "git" | "non_git";
export type TraceDetailLevel = "basic" | "verbose";

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

export type ChangeSetCapability = {
  can_commit: boolean;
  can_discard: boolean;
  can_export: boolean;
  can_export_patch?: boolean;
  reason?: string;
};

export const PERMISSION_MODE_IDS: PermissionMode[] = [
  "default",
  "acceptEdits",
  "plan",
  "dontAsk",
  "bypassPermissions"
];

export const PRIMARY_PERMISSION_MODE_IDS: PermissionMode[] = ["default", "plan"];

export function isPermissionMode(value: string): value is PermissionMode {
  const normalized = value.trim();
  return (PERMISSION_MODE_IDS as string[]).includes(normalized);
}

export function normalizePermissionMode(value: string): PermissionMode {
  return isPermissionMode(value) ? value : "default";
}

export function isDangerousPermissionMode(value: PermissionMode): boolean {
  return value === "dontAsk" || value === "bypassPermissions";
}
