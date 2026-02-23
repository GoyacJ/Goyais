import { getControlClient } from "@/shared/services/clients";
import { withApiFallback } from "@/shared/services/fallback";
import { createMockId, mockData } from "@/shared/services/mockData";
import type {
  CreateWorkspaceRequest,
  ListEnvelope,
  LoginRequest,
  LoginResponse,
  WorkspaceConnectionResult,
  Workspace
} from "@/shared/types/api";

export async function listWorkspaces(): Promise<ListEnvelope<Workspace>> {
  return withApiFallback(
    "workspace.list",
    () => getControlClient().get<ListEnvelope<Workspace>>("/v1/workspaces"),
    () => ({
      items: [...mockData.workspaces],
      next_cursor: null
    })
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
