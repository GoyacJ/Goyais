import { normalizeHttpError, normalizeUnknownError } from "@/lib/api-error";

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
