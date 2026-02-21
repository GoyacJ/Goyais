import { normalizeHttpError, normalizeUnknownError } from "@/lib/api-error";
import type { ModelCatalogResponse, ProviderKey } from "@/types/modelCatalog";

export interface BootstrapStatusResponse {
  setup_mode: boolean;
  allow_public_signup: boolean;
  message: string;
}

export interface HealthResponse {
  status: string;
  version: string;
}

export interface LoginResponse {
  token: string;
  user: {
    user_id: string;
    email: string;
    display_name: string;
  };
}

export interface BootstrapAdminResponse extends LoginResponse {
  workspace: {
    workspace_id: string;
    name: string;
    slug: string;
  };
}

export interface MeResponse {
  user: {
    user_id: string;
    email: string;
    display_name: string;
  };
  memberships: Array<{
    workspace_id: string;
    workspace_name: string;
    workspace_slug: string;
    role_name: string;
  }>;
}

export interface WorkspacesResponse {
  workspaces: Array<{
    workspace_id: string;
    name: string;
    slug: string;
    role_name: string;
  }>;
}

export interface NavigationMenuResponse {
  menu_id: string;
  route: string | null;
  icon_key: string | null;
  i18n_key: string;
  children: NavigationMenuResponse[];
}

export interface NavigationResponse {
  workspace_id: string;
  menus: NavigationMenuResponse[];
  permissions: string[];
  feature_flags: Record<string, boolean>;
}

export interface HubProject {
  project_id: string;
  workspace_id: string;
  name: string;
  root_uri?: string;
  repo_url?: string;
  branch: string;
  auth_ref?: string;
  repo_cache_path?: string;
  sync_status: "pending" | "syncing" | "ready" | "error";
  sync_error?: string;
  last_synced_at?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface HubProjectsResponse {
  projects: HubProject[];
}

export interface HubProjectResponse {
  project: HubProject;
}

export interface HubModelConfig {
  model_config_id: string;
  workspace_id: string;
  provider: ProviderKey;
  model: string;
  base_url: string | null;
  temperature: number;
  max_tokens: number | null;
  secret_ref: string;
  created_at: string;
  updated_at: string;
}

export interface HubModelConfigsResponse {
  model_configs: HubModelConfig[];
}

export interface HubModelConfigResponse {
  model_config: HubModelConfig;
}

export interface HubDeleteResponse {
  ok: true;
}

// ─── Skills / MCP ────────────────────────────────────────────────────────────

export interface HubSkillSet {
  skill_set_id: string;
  workspace_id: string;
  name: string;
  description?: string;
  created_by: string;
  created_at: string;
}

export interface HubSkill {
  skill_id: string;
  skill_set_id: string;
  name: string;
  type: "tool_combo" | "template" | "custom";
  config_json: string;
  created_at: string;
}

export interface HubMCPConnector {
  connector_id: string;
  workspace_id: string;
  name: string;
  transport: "stdio" | "sse" | "streamable_http";
  endpoint: string;
  secret_ref?: string;
  config_json: string;
  enabled: boolean;
  created_by: string;
  created_at: string;
}

function normalizeServerUrl(serverUrl: string): string {
  return serverUrl.trim().replace(/\/+$/, "");
}

async function requestJson<T>(
  serverUrl: string,
  path: string,
  init?: RequestInit,
  bearerToken?: string
): Promise<T> {
  try {
    const headers = new Headers(init?.headers ?? {});
    headers.set("X-Trace-Id", crypto.randomUUID());

    if (bearerToken) {
      headers.set("Authorization", `Bearer ${bearerToken}`);
    }

    const response = await fetch(`${normalizeServerUrl(serverUrl)}${path}`, {
      ...init,
      headers
    });

    if (!response.ok) {
      throw await normalizeHttpError(response);
    }

    return (await response.json()) as T;
  } catch (error) {
    throw normalizeUnknownError(error);
  }
}

async function requestText(
  serverUrl: string,
  path: string,
  init?: RequestInit,
  bearerToken?: string
): Promise<string> {
  try {
    const headers = new Headers(init?.headers ?? {});
    headers.set("X-Trace-Id", crypto.randomUUID());

    if (bearerToken) {
      headers.set("Authorization", `Bearer ${bearerToken}`);
    }

    const response = await fetch(`${normalizeServerUrl(serverUrl)}${path}`, {
      ...init,
      headers
    });

    if (!response.ok) {
      throw await normalizeHttpError(response);
    }

    return await response.text();
  } catch (error) {
    throw normalizeUnknownError(error);
  }
}

export async function getBootstrapStatus(serverUrl: string): Promise<BootstrapStatusResponse> {
  return requestJson<BootstrapStatusResponse>(serverUrl, "/v1/auth/bootstrap/status");
}

export async function getHealth(serverUrl: string): Promise<HealthResponse> {
  return requestJson<HealthResponse>(serverUrl, "/v1/health");
}

export async function bootstrapAdmin(
  serverUrl: string,
  payload: {
    bootstrap_token: string;
    email: string;
    password: string;
    display_name: string;
  }
): Promise<BootstrapAdminResponse> {
  return requestJson<BootstrapAdminResponse>(serverUrl, "/v1/auth/bootstrap/admin", {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(payload)
  });
}

export async function login(
  serverUrl: string,
  payload: {
    email: string;
    password: string;
  }
): Promise<LoginResponse> {
  return requestJson<LoginResponse>(serverUrl, "/v1/auth/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(payload)
  });
}

export async function me(serverUrl: string, token: string): Promise<MeResponse> {
  return requestJson<MeResponse>(serverUrl, "/v1/me", undefined, token);
}

export async function listWorkspaces(serverUrl: string, token: string): Promise<WorkspacesResponse> {
  return requestJson<WorkspacesResponse>(serverUrl, "/v1/workspaces", undefined, token);
}

export async function getNavigation(
  serverUrl: string,
  token: string,
  workspaceId: string
): Promise<NavigationResponse> {
  const query = encodeURIComponent(workspaceId);
  return requestJson<NavigationResponse>(serverUrl, `/v1/me/navigation?workspace_id=${query}`, undefined, token);
}

export async function listProjects(
  serverUrl: string,
  token: string,
  workspaceId: string
): Promise<HubProjectsResponse> {
  const query = encodeURIComponent(workspaceId);
  return requestJson<HubProjectsResponse>(serverUrl, `/v1/projects?workspace_id=${query}`, undefined, token);
}

export async function createProject(
  serverUrl: string,
  token: string,
  workspaceId: string,
  payload: {
    name: string;
    root_uri?: string;
    repo_url?: string;
    branch?: string;
    auth_ref?: string;
  }
): Promise<HubProjectResponse> {
  const query = encodeURIComponent(workspaceId);
  return requestJson<HubProjectResponse>(
    serverUrl,
    `/v1/projects?workspace_id=${query}`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify(payload)
    },
    token
  );
}

export async function deleteProject(
  serverUrl: string,
  token: string,
  workspaceId: string,
  projectId: string
): Promise<HubDeleteResponse> {
  const query = encodeURIComponent(workspaceId);
  const encodedProjectId = encodeURIComponent(projectId);
  return requestJson<HubDeleteResponse>(
    serverUrl,
    `/v1/projects/${encodedProjectId}?workspace_id=${query}`,
    {
      method: "DELETE"
    },
    token
  );
}

export async function syncProject(
  serverUrl: string,
  token: string,
  workspaceId: string,
  projectId: string
): Promise<{ status: string }> {
  const query = encodeURIComponent(workspaceId);
  const encodedProjectId = encodeURIComponent(projectId);
  return requestJson<{ status: string }>(
    serverUrl,
    `/v1/projects/${encodedProjectId}/sync?workspace_id=${query}`,
    { method: "POST" },
    token
  );
}

export async function listModelConfigs(
  serverUrl: string,
  token: string,
  workspaceId: string
): Promise<HubModelConfigsResponse> {
  const query = encodeURIComponent(workspaceId);
  return requestJson<HubModelConfigsResponse>(serverUrl, `/v1/model-configs?workspace_id=${query}`, undefined, token);
}

export async function createModelConfig(
  serverUrl: string,
  token: string,
  workspaceId: string,
  payload: {
    provider: ProviderKey;
    model: string;
    base_url?: string | null;
    temperature?: number;
    max_tokens?: number | null;
    api_key: string;
  }
): Promise<HubModelConfigResponse> {
  const query = encodeURIComponent(workspaceId);
  return requestJson<HubModelConfigResponse>(
    serverUrl,
    `/v1/model-configs?workspace_id=${query}`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify(payload)
    },
    token
  );
}

export async function updateModelConfig(
  serverUrl: string,
  token: string,
  workspaceId: string,
  modelConfigId: string,
  payload: {
    provider?: ProviderKey;
    model?: string;
    base_url?: string | null;
    temperature?: number;
    max_tokens?: number | null;
    api_key?: string;
  }
): Promise<HubModelConfigResponse> {
  const query = encodeURIComponent(workspaceId);
  const encodedModelConfigId = encodeURIComponent(modelConfigId);
  return requestJson<HubModelConfigResponse>(
    serverUrl,
    `/v1/model-configs/${encodedModelConfigId}?workspace_id=${query}`,
    {
      method: "PUT",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify(payload)
    },
    token
  );
}

export async function deleteModelConfig(
  serverUrl: string,
  token: string,
  workspaceId: string,
  modelConfigId: string
): Promise<HubDeleteResponse> {
  const query = encodeURIComponent(workspaceId);
  const encodedModelConfigId = encodeURIComponent(modelConfigId);
  return requestJson<HubDeleteResponse>(
    serverUrl,
    `/v1/model-configs/${encodedModelConfigId}?workspace_id=${query}`,
    {
      method: "DELETE"
    },
    token
  );
}

export async function listRuntimeModelCatalog(
  serverUrl: string,
  token: string,
  workspaceId: string,
  modelConfigId: string
): Promise<ModelCatalogResponse> {
  const query = encodeURIComponent(workspaceId);
  const encodedModelConfigId = encodeURIComponent(modelConfigId);
  return requestJson<ModelCatalogResponse>(
    serverUrl,
    `/v1/runtime/model-configs/${encodedModelConfigId}/models?workspace_id=${query}`,
    undefined,
    token
  );
}

// ============================================================
// Phase 1: Session API
// ============================================================

export interface HubSession {
  session_id: string;
  workspace_id: string;
  project_id: string;
  title: string;
  mode: "plan" | "agent";
  model_config_id: string | null;
  skill_set_ids: string;
  mcp_connector_ids: string;
  use_worktree: boolean;
  active_execution_id: string | null;
  status: "idle" | "executing" | "waiting_confirmation";
  created_by: string;
  created_at: string;
  updated_at: string;
  archived_at: string | null;
}

export interface HubSessionsResponse {
  sessions: HubSession[];
}

export interface HubSessionResponse {
  session: HubSession;
}

export interface CreateSessionPayload {
  project_id: string;
  title?: string;
  mode?: "plan" | "agent";
  model_config_id?: string | null;
  use_worktree?: boolean;
}

export interface UpdateSessionPayload {
  title?: string;
  mode?: "plan" | "agent";
  model_config_id?: string | null;
}

export async function listSessions(
  serverUrl: string,
  token: string,
  workspaceId: string,
  projectId: string
): Promise<HubSessionsResponse> {
  const wsQ = encodeURIComponent(workspaceId);
  const projQ = encodeURIComponent(projectId);
  return requestJson<HubSessionsResponse>(
    serverUrl,
    `/v1/sessions?workspace_id=${wsQ}&project_id=${projQ}`,
    undefined,
    token
  );
}

export async function createSession(
  serverUrl: string,
  token: string,
  workspaceId: string,
  payload: CreateSessionPayload
): Promise<HubSessionResponse> {
  const wsQ = encodeURIComponent(workspaceId);
  return requestJson<HubSessionResponse>(
    serverUrl,
    `/v1/sessions?workspace_id=${wsQ}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload)
    },
    token
  );
}

export async function updateSession(
  serverUrl: string,
  token: string,
  workspaceId: string,
  sessionId: string,
  payload: UpdateSessionPayload
): Promise<HubSessionResponse> {
  const wsQ = encodeURIComponent(workspaceId);
  const sid = encodeURIComponent(sessionId);
  return requestJson<HubSessionResponse>(
    serverUrl,
    `/v1/sessions/${sid}?workspace_id=${wsQ}`,
    {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload)
    },
    token
  );
}

export async function archiveSession(
  serverUrl: string,
  token: string,
  workspaceId: string,
  sessionId: string
): Promise<void> {
  const wsQ = encodeURIComponent(workspaceId);
  const sid = encodeURIComponent(sessionId);
  await requestJson<void>(
    serverUrl,
    `/v1/sessions/${sid}?workspace_id=${wsQ}`,
    { method: "DELETE" },
    token
  );
}

// ============================================================
// Phase 2: Execution API
// ============================================================

export interface ExecutionInfo {
  execution_id: string;
  trace_id: string;
  session_id: string;
  state: "pending" | "executing" | "waiting_confirmation" | "completed" | "failed" | "cancelled";
}

export interface ExecutionResponse {
  execution_id: string;
  trace_id: string;
  session_id: string;
  state: string;
}

export interface SessionBusyError {
  code: "E_SESSION_BUSY";
  active_execution_id: string;
  session_id: string;
}

/** POST /v1/sessions/{id}/execute — returns 202 or 409 SESSION_BUSY */
export async function executeSession(
  serverUrl: string,
  token: string,
  workspaceId: string,
  sessionId: string,
  message: string
): Promise<ExecutionResponse> {
  const wsQ = encodeURIComponent(workspaceId);
  const sid = encodeURIComponent(sessionId);
  return requestJson<ExecutionResponse>(
    serverUrl,
    `/v1/sessions/${sid}/execute?workspace_id=${wsQ}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ message })
    },
    token
  );
}

/** DELETE /v1/executions/{id}/cancel */
export async function cancelExecution(
  serverUrl: string,
  token: string,
  workspaceId: string,
  executionId: string
): Promise<void> {
  const wsQ = encodeURIComponent(workspaceId);
  const eid = encodeURIComponent(executionId);
  await requestJson<void>(
    serverUrl,
    `/v1/executions/${eid}/cancel?workspace_id=${wsQ}`,
    { method: "DELETE" },
    token
  );
}

/** POST /v1/confirmations — approve or deny a tool call */
export async function decideConfirmation(
  serverUrl: string,
  token: string,
  workspaceId: string,
  executionId: string,
  callId: string,
  decision: "approved" | "denied"
): Promise<void> {
  const wsQ = encodeURIComponent(workspaceId);
  await requestJson<void>(
    serverUrl,
    `/v1/confirmations?workspace_id=${wsQ}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ execution_id: executionId, call_id: callId, decision })
    },
    token
  );
}

export interface ExecutionCommitResponse {
  commit_sha: string;
}

/** POST /v1/executions/{id}/commit */
export async function commitExecution(
  serverUrl: string,
  token: string,
  workspaceId: string,
  executionId: string,
  message?: string
): Promise<ExecutionCommitResponse> {
  const wsQ = encodeURIComponent(workspaceId);
  const eid = encodeURIComponent(executionId);
  return requestJson<ExecutionCommitResponse>(
    serverUrl,
    `/v1/executions/${eid}/commit?workspace_id=${wsQ}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ message: message?.trim() || undefined })
    },
    token
  );
}

/** GET /v1/executions/{id}/patch */
export async function exportExecutionPatch(
  serverUrl: string,
  token: string,
  workspaceId: string,
  executionId: string
): Promise<string> {
  const wsQ = encodeURIComponent(workspaceId);
  const eid = encodeURIComponent(executionId);
  return requestText(
    serverUrl,
    `/v1/executions/${eid}/patch?workspace_id=${wsQ}`,
    { method: "GET" },
    token
  );
}

/** DELETE /v1/executions/{id}/discard */
export async function discardExecution(
  serverUrl: string,
  token: string,
  workspaceId: string,
  executionId: string
): Promise<void> {
  const wsQ = encodeURIComponent(workspaceId);
  const eid = encodeURIComponent(executionId);
  await requestJson<void>(
    serverUrl,
    `/v1/executions/${eid}/discard?workspace_id=${wsQ}`,
    { method: "DELETE" },
    token
  );
}

/**
 * Subscribe to SSE events for a session's active execution.
 * Returns a cleanup function.
 */
// ─── Skill Set API ────────────────────────────────────────────────────────────

export async function listSkillSets(serverUrl: string, token: string, workspaceId: string): Promise<{ skill_sets: HubSkillSet[] }> {
  const q = encodeURIComponent(workspaceId);
  return requestJson(serverUrl, `/v1/skill-sets?workspace_id=${q}`, undefined, token);
}

export async function createSkillSet(
  serverUrl: string,
  token: string,
  workspaceId: string,
  payload: { name: string; description?: string }
): Promise<{ skill_set: HubSkillSet }> {
  const q = encodeURIComponent(workspaceId);
  return requestJson(serverUrl, `/v1/skill-sets?workspace_id=${q}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  }, token);
}

export async function updateSkillSet(
  serverUrl: string,
  token: string,
  workspaceId: string,
  skillSetId: string,
  payload: { name?: string; description?: string }
): Promise<{ skill_set: HubSkillSet }> {
  const q = encodeURIComponent(workspaceId);
  const id = encodeURIComponent(skillSetId);
  return requestJson(serverUrl, `/v1/skill-sets/${id}?workspace_id=${q}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  }, token);
}

export async function deleteSkillSet(
  serverUrl: string,
  token: string,
  workspaceId: string,
  skillSetId: string
): Promise<void> {
  const q = encodeURIComponent(workspaceId);
  const id = encodeURIComponent(skillSetId);
  await requestJson(serverUrl, `/v1/skill-sets/${id}?workspace_id=${q}`, { method: "DELETE" }, token);
}

export async function listSkills(serverUrl: string, token: string, workspaceId: string, skillSetId: string): Promise<{ skills: HubSkill[] }> {
  const q = encodeURIComponent(workspaceId);
  const id = encodeURIComponent(skillSetId);
  return requestJson(serverUrl, `/v1/skill-sets/${id}/skills?workspace_id=${q}`, undefined, token);
}

export async function createSkill(
  serverUrl: string,
  token: string,
  workspaceId: string,
  skillSetId: string,
  payload: { name: string; type: string; config_json?: string }
): Promise<{ skill: HubSkill }> {
  const q = encodeURIComponent(workspaceId);
  const id = encodeURIComponent(skillSetId);
  return requestJson(serverUrl, `/v1/skill-sets/${id}/skills?workspace_id=${q}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  }, token);
}

export async function deleteSkill(serverUrl: string, token: string, workspaceId: string, skillId: string): Promise<void> {
  const q = encodeURIComponent(workspaceId);
  const id = encodeURIComponent(skillId);
  await requestJson(serverUrl, `/v1/skills/${id}?workspace_id=${q}`, { method: "DELETE" }, token);
}

// ─── MCP Connector API ───────────────────────────────────────────────────────

export async function listMCPConnectors(serverUrl: string, token: string, workspaceId: string): Promise<{ mcp_connectors: HubMCPConnector[] }> {
  const q = encodeURIComponent(workspaceId);
  return requestJson(serverUrl, `/v1/mcp-connectors?workspace_id=${q}`, undefined, token);
}

export async function createMCPConnector(
  serverUrl: string,
  token: string,
  workspaceId: string,
  payload: { name: string; transport: string; endpoint: string; secret_ref?: string }
): Promise<{ mcp_connector: HubMCPConnector }> {
  const q = encodeURIComponent(workspaceId);
  return requestJson(serverUrl, `/v1/mcp-connectors?workspace_id=${q}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  }, token);
}

export async function updateMCPConnector(
  serverUrl: string,
  token: string,
  workspaceId: string,
  connectorId: string,
  payload: { name?: string; transport?: string; endpoint?: string; secret_ref?: string; enabled?: boolean }
): Promise<{ mcp_connector: HubMCPConnector }> {
  const q = encodeURIComponent(workspaceId);
  const id = encodeURIComponent(connectorId);
  return requestJson(serverUrl, `/v1/mcp-connectors/${id}?workspace_id=${q}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  }, token);
}

export async function deleteMCPConnector(
  serverUrl: string,
  token: string,
  workspaceId: string,
  connectorId: string
): Promise<void> {
  const q = encodeURIComponent(workspaceId);
  const id = encodeURIComponent(connectorId);
  await requestJson(serverUrl, `/v1/mcp-connectors/${id}?workspace_id=${q}`, { method: "DELETE" }, token);
}

export function subscribeSessionEvents(
  serverUrl: string,
  token: string,
  workspaceId: string,
  sessionId: string,
  sinceSeq: number,
  onEvent: (type: string, payloadJson: string, seq: number) => void,
  onError?: (err: Error) => void
): () => void {
  const wsQ = encodeURIComponent(workspaceId);
  const sid = encodeURIComponent(sessionId);
  const url = `${normalizeServerUrl(serverUrl)}/v1/sessions/${sid}/events?workspace_id=${wsQ}&since_seq=${sinceSeq}`;

  // Use fetch + ReadableStream for SSE with Authorization header
  const controller = new AbortController();

  (async () => {
    try {
      const resp = await fetch(url, {
        headers: { Authorization: `Bearer ${token}`, "X-Trace-Id": crypto.randomUUID() },
        signal: controller.signal
      });
      if (!resp.ok || !resp.body) {
        onError?.(new Error(`SSE connect failed: ${resp.status}`));
        return;
      }
      const reader = resp.body.getReader();
      const decoder = new TextDecoder();
      let buf = "";
      let curId = 0;
      let curType = "message";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buf += decoder.decode(value, { stream: true });
        const lines = buf.split("\n");
        buf = lines.pop() ?? "";
        for (const line of lines) {
          if (line.startsWith("id:")) {
            curId = parseInt(line.slice(3).trim(), 10);
          } else if (line.startsWith("event:")) {
            curType = line.slice(6).trim();
          } else if (line.startsWith("data:")) {
            const data = line.slice(5).trim();
            onEvent(curType, data, curId);
            curType = "message";
          }
        }
      }
    } catch (err) {
      if ((err as Error).name !== "AbortError") {
        onError?.(err as Error);
      }
    }
  })();

  return () => controller.abort();
}
