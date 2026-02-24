import { reactive } from "vue";

import type {
  AdminAuditEvent,
  AdminRole,
  AdminUser,
  Conversation,
  Project,
  Resource,
  ResourceConfig,
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

const defaultProjects: Project[] = [
  {
    id: "proj_local_main",
    workspace_id: localWorkspace.id,
    name: "Goyais Desktop",
    repo_path: "/Users/goya/Repo/Git/Goyais",
    is_git: true,
    default_mode: "agent",
    default_model_id: "gpt-5.3",
    current_revision: 0,
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
    model_id: "gpt-5.3",
    base_revision: 0,
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
    model_id: "gpt-5-mini",
    base_revision: 0,
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
    name: "gpt-5.3",
    source: "workspace_native",
    scope: "private",
    share_status: "approved",
    owner_user_id: "local_user",
    enabled: true,
    created_at: now,
    updated_at: now
  }
];

const defaultResourceConfigs: ResourceConfig[] = [
  {
    id: "rc_model_1",
    workspace_id: localWorkspace.id,
    type: "model",
    name: "OpenAI Default",
    enabled: true,
    model: {
      vendor: "OpenAI",
      model_id: "gpt-5.3",
      api_key_masked: "sk-****",
      base_url: "https://api.openai.com/v1"
    },
    created_at: now,
    updated_at: now
  },
  {
    id: "rc_rule_1",
    workspace_id: localWorkspace.id,
    type: "rule",
    name: "Security Rule",
    enabled: true,
    rule: { content: "## Security\\n- Never expose secrets." },
    created_at: now,
    updated_at: now
  },
  {
    id: "rc_skill_1",
    workspace_id: localWorkspace.id,
    type: "skill",
    name: "Review Skill",
    enabled: true,
    skill: { content: "## Review\\n- Focus on regressions and tests." },
    created_at: now,
    updated_at: now
  },
  {
    id: "rc_mcp_1",
    workspace_id: localWorkspace.id,
    type: "mcp",
    name: "Filesystem MCP",
    enabled: true,
    mcp: {
      transport: "stdio",
      command: "npx @modelcontextprotocol/server-filesystem",
      status: "connected",
      tools: ["files.read", "files.write"]
    },
    created_at: now,
    updated_at: now
  },
  {
    id: "rc_mcp_2",
    workspace_id: localWorkspace.id,
    type: "mcp",
    name: "GitHub MCP",
    enabled: true,
    mcp: {
      transport: "http_sse",
      endpoint: "http://127.0.0.1:9001/sse",
      status: "connected",
      tools: ["repos.search", "issues.list", "pull_requests.list"],
      last_connected_at: now
    },
    created_at: now,
    updated_at: now
  }
];

const defaultUsers: AdminUser[] = [];

const defaultRoles: AdminRole[] = [
  { key: "viewer", name: "Viewer", permissions: ["read"], enabled: true },
  { key: "developer", name: "Developer", permissions: ["read", "write", "execute"], enabled: true },
  { key: "approver", name: "Approver", permissions: ["read", "approve"], enabled: true },
  { key: "admin", name: "Admin", permissions: ["*"], enabled: true }
];

const defaultAuditEvents: AdminAuditEvent[] = [
  {
    id: "audit_1",
    actor: "system",
    action: "workspace.seed_local",
    resource: localWorkspace.id,
    result: "success",
    trace_id: "tr_mock_001",
    timestamp: now
  }
];

const defaultShareRequests: ShareRequest[] = [];

export const mockData = reactive({
  workspaces: [localWorkspace] as Workspace[],
  projects: [...defaultProjects] as Project[],
  conversations: [...defaultConversations] as Conversation[],
  resources: [...defaultResources] as Resource[],
  resourceConfigs: [...defaultResourceConfigs] as ResourceConfig[],
  users: [...defaultUsers] as AdminUser[],
  roles: [...defaultRoles] as AdminRole[],
  auditEvents: [...defaultAuditEvents] as AdminAuditEvent[],
  shareRequests: [...defaultShareRequests] as ShareRequest[]
});

export function createMockId(prefix: string): string {
  const randomPart = Math.random().toString(16).slice(2, 8);
  return `${prefix}_${randomPart}`;
}
