import { reactive } from "vue";

import type {
  AdminAuditEvent,
  AdminRole,
  AdminUser,
  Conversation,
  Project,
  Resource,
  ShareRequest,
  Workspace
} from "@/shared/types/api";

const now = new Date().toISOString();

const localWorkspace: Workspace = {
  id: "ws_local",
  name: "Local Workspace",
  mode: "local",
  hub_url: null,
  is_default_local: true,
  created_at: now,
  login_disabled: true,
  auth_mode: "disabled"
};

const remoteWorkspace: Workspace = {
  id: "ws_remote_demo",
  name: "Remote · Demo Hub",
  mode: "remote",
  hub_url: "http://127.0.0.1:8787",
  is_default_local: false,
  created_at: now,
  login_disabled: false,
  auth_mode: "password_or_token"
};

const defaultProjects: Project[] = [
  {
    id: "proj_local_main",
    workspace_id: localWorkspace.id,
    name: "Goyais Desktop",
    repo_path: "/Users/goya/Repo/Git/Goyais",
    is_git: true,
    default_mode: "agent",
    default_model_id: "gpt-4.1",
    created_at: now,
    updated_at: now
  },
  {
    id: "proj_remote_ops",
    workspace_id: remoteWorkspace.id,
    name: "Remote Ops",
    repo_path: "/srv/workspaces/remote-ops",
    is_git: false,
    default_mode: "plan",
    default_model_id: "gpt-4.1-mini",
    created_at: now,
    updated_at: now
  }
];

const defaultConversations: Conversation[] = [
  {
    id: "conv_1",
    workspace_id: localWorkspace.id,
    project_id: "proj_local_main",
    name: "主会话 / Main",
    queue_state: "idle",
    default_mode: "agent",
    model_id: "gpt-4.1",
    active_execution_id: null,
    created_at: now,
    updated_at: now
  },
  {
    id: "conv_2",
    workspace_id: localWorkspace.id,
    project_id: "proj_local_main",
    name: "计划验证 / Plan Review",
    queue_state: "idle",
    default_mode: "plan",
    model_id: "gpt-4.1-mini",
    active_execution_id: null,
    created_at: now,
    updated_at: now
  },
  {
    id: "conv_remote_1",
    workspace_id: remoteWorkspace.id,
    project_id: "proj_remote_ops",
    name: "Remote Conversation",
    queue_state: "idle",
    default_mode: "agent",
    model_id: "gpt-4.1",
    active_execution_id: null,
    created_at: now,
    updated_at: now
  }
];

const defaultResources: Resource[] = [
  {
    id: "res_model_1",
    workspace_id: localWorkspace.id,
    type: "model",
    name: "gpt-4.1",
    source: "workspace_native",
    scope: "private",
    share_status: "approved",
    owner_user_id: "local_user",
    enabled: true,
    created_at: now,
    updated_at: now
  },
  {
    id: "res_rule_1",
    workspace_id: remoteWorkspace.id,
    type: "rule",
    name: "Safety Guardrails",
    source: "local_import",
    scope: "shared",
    share_status: "approved",
    owner_user_id: "u_admin",
    enabled: true,
    created_at: now,
    updated_at: now
  },
  {
    id: "res_mcp_1",
    workspace_id: remoteWorkspace.id,
    type: "mcp",
    name: "Git MCP",
    source: "workspace_native",
    scope: "private",
    share_status: "pending",
    owner_user_id: "u_dev",
    enabled: true,
    created_at: now,
    updated_at: now
  }
];

const defaultUsers: AdminUser[] = [
  {
    id: "u_admin",
    workspace_id: remoteWorkspace.id,
    username: "admin",
    display_name: "Remote Admin",
    role: "admin",
    enabled: true,
    created_at: now
  },
  {
    id: "u_dev",
    workspace_id: remoteWorkspace.id,
    username: "developer",
    display_name: "Remote Developer",
    role: "developer",
    enabled: true,
    created_at: now
  }
];

const defaultRoles: AdminRole[] = [
  { key: "viewer", name: "Viewer", permissions: ["read"], enabled: true },
  { key: "developer", name: "Developer", permissions: ["read", "write", "execute"], enabled: true },
  { key: "approver", name: "Approver", permissions: ["read", "approve"], enabled: true },
  { key: "admin", name: "Admin", permissions: ["*"], enabled: true }
];

const defaultAuditEvents: AdminAuditEvent[] = [
  {
    id: "audit_1",
    actor: "admin",
    action: "permissions.update",
    resource: "menu_bindings",
    result: "success",
    trace_id: "tr_mock_001",
    timestamp: now
  },
  {
    id: "audit_2",
    actor: "developer",
    action: "share.approve",
    resource: "share_request_1",
    result: "denied",
    trace_id: "tr_mock_002",
    timestamp: now
  }
];

const defaultShareRequests: ShareRequest[] = [
  {
    id: "share_request_1",
    workspace_id: remoteWorkspace.id,
    resource_id: "res_mcp_1",
    status: "pending",
    requester_user_id: "u_dev",
    created_at: now,
    updated_at: now
  }
];

export const mockData = reactive({
  workspaces: [localWorkspace, remoteWorkspace] as Workspace[],
  projects: [...defaultProjects] as Project[],
  conversations: [...defaultConversations] as Conversation[],
  resources: [...defaultResources] as Resource[],
  users: [...defaultUsers] as AdminUser[],
  roles: [...defaultRoles] as AdminRole[],
  auditEvents: [...defaultAuditEvents] as AdminAuditEvent[],
  shareRequests: [...defaultShareRequests] as ShareRequest[]
});

export function createMockId(prefix: string): string {
  const randomPart = Math.random().toString(16).slice(2, 8);
  return `${prefix}_${randomPart}`;
}
