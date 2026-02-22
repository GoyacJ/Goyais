import { getControlClient } from "@/shared/services/clients";
import type {
  CreateWorkspaceRequest,
  ListEnvelope,
  LoginRequest,
  LoginResponse,
  Workspace
} from "@/shared/types/api";

export async function listWorkspaces(): Promise<ListEnvelope<Workspace>> {
  return getControlClient().get<ListEnvelope<Workspace>>("/v1/workspaces");
}

export async function createRemoteWorkspace(input: CreateWorkspaceRequest): Promise<Workspace> {
  return getControlClient().post<Workspace>("/v1/workspaces", input);
}

export async function loginWorkspace(input: LoginRequest): Promise<LoginResponse> {
  return getControlClient().post<LoginResponse>("/v1/auth/login", input);
}
