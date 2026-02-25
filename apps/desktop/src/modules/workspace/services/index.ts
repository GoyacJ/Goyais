import { getControlClient } from "@/shared/services/clients";
import type {
  CreateWorkspaceRequest,
  ListEnvelope,
  LoginRequest,
  LoginResponse,
  LogoutRequest,
  PaginationQuery,
  WorkspaceStatusResponse,
  RefreshRequest,
  WorkspaceConnectionResult,
  Workspace
} from "@/shared/types/api";

export async function listWorkspaces(query: PaginationQuery = {}): Promise<ListEnvelope<Workspace>> {
  const search = buildPaginationSearch(query);
  return getControlClient().get<ListEnvelope<Workspace>>(`/v1/workspaces${search}`);
}

export async function createRemoteConnection(input: CreateWorkspaceRequest): Promise<WorkspaceConnectionResult> {
  return getControlClient().post<WorkspaceConnectionResult>("/v1/workspaces/remote-connections", input);
}

export async function getWorkspaceStatus(
  workspaceId: string,
  options: {
    conversationId?: string;
    token?: string;
  } = {}
): Promise<WorkspaceStatusResponse> {
  const normalizedWorkspaceID = workspaceId.trim();
  if (normalizedWorkspaceID === "") {
    throw new Error("workspace_id is required");
  }

  const params = new URLSearchParams();
  const conversationID = options.conversationId?.trim() ?? "";
  if (conversationID !== "") {
    params.set("conversation_id", conversationID);
  }
  const query = params.toString() === "" ? "" : `?${params.toString()}`;

  return getControlClient().get<WorkspaceStatusResponse>(`/v1/workspaces/${normalizedWorkspaceID}/status${query}`, {
    token: options.token?.trim() === "" ? undefined : options.token
  });
}

export async function createRemoteWorkspace(input: { name: string; hub_url: string }): Promise<Workspace> {
  return getControlClient().post<Workspace>("/v1/workspaces", {
    name: input.name,
    hub_url: input.hub_url
  });
}

export async function loginWorkspace(input: LoginRequest): Promise<LoginResponse> {
  return getControlClient().post<LoginResponse>("/v1/auth/login", input);
}

export async function refreshWorkspaceSession(input: RefreshRequest): Promise<LoginResponse> {
  return getControlClient().post<LoginResponse>("/v1/auth/refresh", input);
}

export async function logoutWorkspaceSession(input: LogoutRequest): Promise<{ ok: true }> {
  return getControlClient().post<{ ok: true }>("/v1/auth/logout", input);
}

function buildPaginationSearch(query: PaginationQuery): string {
  const params = new URLSearchParams();
  if (query.cursor) {
    params.set("cursor", query.cursor);
  }
  if (query.limit !== undefined) {
    params.set("limit", String(query.limit));
  }
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}
