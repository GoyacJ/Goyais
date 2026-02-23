import { getControlClient } from "@/shared/services/clients";
import { withApiFallback } from "@/shared/services/fallback";
import { createMockId, mockData } from "@/shared/services/mockData";
import type {
  CreateWorkspaceRequest,
  ListEnvelope,
  PaginationQuery,
  LoginRequest,
  LoginResponse,
  WorkspaceConnectionResult,
  Workspace
} from "@/shared/types/api";

export async function listWorkspaces(query: PaginationQuery = {}): Promise<ListEnvelope<Workspace>> {
  const search = buildPaginationSearch(query);
  return withApiFallback(
    "workspace.list",
    () => getControlClient().get<ListEnvelope<Workspace>>(`/v1/workspaces${search}`),
    () => paginateMock([...mockData.workspaces], query)
  );
}

export async function createRemoteConnection(input: CreateWorkspaceRequest): Promise<WorkspaceConnectionResult> {
  return withApiFallback(
    "workspace.createRemoteConnection",
    () => getControlClient().post<WorkspaceConnectionResult>("/v1/workspaces/remote-connections", input),
    () => {
      const now = new Date().toISOString();
      const hostName = input.hub_url.trim().replace(/^https?:\/\//, "").split("/")[0] ?? "remote";
      const created: Workspace = {
        id: createMockId("ws_remote"),
        name: input.name?.trim() || `Remote · ${hostName}`,
        mode: "remote",
        hub_url: input.hub_url.trim(),
        is_default_local: false,
        created_at: now,
        login_disabled: input.login_disabled ?? false,
        auth_mode: input.auth_mode ?? "password_or_token"
      };
      mockData.workspaces.push(created);
      return {
        workspace: created,
        connection: {
          workspace_id: created.id,
          hub_url: created.hub_url ?? "",
          username: input.username,
          connection_status: "connected",
          connected_at: now,
          access_token: `at_${createMockId("remote")}`
        },
        access_token: `at_${createMockId("remote")}`
      };
    }
  );
}

export async function createRemoteWorkspace(input: { name: string; hub_url: string }): Promise<Workspace> {
  return withApiFallback(
    "workspace.createRemote",
    () =>
      getControlClient().post<Workspace>("/v1/workspaces", {
        name: input.name,
        hub_url: input.hub_url
      }),
    () => {
      const hostName = input.hub_url.trim().replace(/^https?:\/\//, "").split("/")[0] ?? "remote";
      const created: Workspace = {
        id: createMockId("ws_remote"),
        name: input.name.trim() || `Remote · ${hostName}`,
        mode: "remote",
        hub_url: input.hub_url.trim(),
        is_default_local: false,
        created_at: new Date().toISOString(),
        login_disabled: false,
        auth_mode: "password_or_token"
      };
      mockData.workspaces.push(created);
      return created;
    }
  );
}

export async function loginWorkspace(input: LoginRequest): Promise<LoginResponse> {
  return withApiFallback(
    "workspace.login",
    () => getControlClient().post<LoginResponse>("/v1/auth/login", input),
    () => ({
      access_token: `at_${createMockId("mock")}`,
      token_type: "bearer"
    })
  );
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

function paginateMock<T>(items: T[], query: PaginationQuery): ListEnvelope<T> {
  const start = Number.parseInt(query.cursor ?? "0", 10);
  const safeStart = Number.isNaN(start) || start < 0 ? 0 : start;
  const limit = query.limit !== undefined && query.limit > 0 ? query.limit : 20;
  const end = Math.min(safeStart + limit, items.length);
  const nextCursor = end < items.length ? String(end) : null;
  return {
    items: items.slice(safeStart, end),
    next_cursor: nextCursor
  };
}
