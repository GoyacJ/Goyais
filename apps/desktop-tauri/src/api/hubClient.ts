import { normalizeHttpError, normalizeUnknownError } from "@/lib/api-error";
import type { ModelCatalogResponse, ProviderKey } from "@/types/modelCatalog";

export interface BootstrapStatusResponse {
  setup_mode: boolean;
  allow_public_signup: boolean;
  message: string;
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
  root_uri: string;
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

export async function getBootstrapStatus(serverUrl: string): Promise<BootstrapStatusResponse> {
  return requestJson<BootstrapStatusResponse>(serverUrl, "/v1/auth/bootstrap/status");
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
    root_uri: string;
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
