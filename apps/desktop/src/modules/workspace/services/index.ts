import { getControlClient } from "@/shared/services/clients";
import { withApiFallback } from "@/shared/services/fallback";
import { createMockId, mockData } from "@/shared/services/mockData";
import type {
  CreateWorkspaceRequest,
  ListEnvelope,
  LoginRequest,
  LoginResponse,
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

export async function createRemoteWorkspace(input: CreateWorkspaceRequest): Promise<Workspace> {
  return withApiFallback(
    "workspace.createRemote",
    () => getControlClient().post<Workspace>("/v1/workspaces", input),
    () => {
      const created: Workspace = {
        id: createMockId("ws_remote"),
        name: input.name.trim(),
        mode: "remote",
        hub_url: input.hub_url.trim(),
        is_default_local: false,
        created_at: new Date().toISOString(),
        login_disabled: input.login_disabled ?? false,
        auth_mode: input.auth_mode ?? "password_or_token"
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
