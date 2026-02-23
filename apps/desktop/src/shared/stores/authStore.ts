import { reactive } from "vue";

import { getControlClient, getTargetClient } from "@/shared/services/clients";
import { ApiError } from "@/shared/services/http";
import { getCurrentWorkspace, workspaceStore } from "@/shared/stores/workspaceStore";
import type { Capabilities, Me } from "@/shared/types/api";

type AuthState = {
  tokensByWorkspaceId: Record<string, string>;
  me: Me | null;
  capabilities: Capabilities;
  loading: boolean;
  error: string;
};

const defaultCapabilities: Capabilities = {
  admin_console: false,
  resource_write: false,
  execution_control: false
};

const initialState: AuthState = {
  tokensByWorkspaceId: {},
  me: null,
  capabilities: { ...defaultCapabilities },
  loading: false,
  error: ""
};

export const authStore = reactive<AuthState>({ ...initialState });

export function resetAuthStore(): void {
  authStore.tokensByWorkspaceId = {};
  authStore.me = null;
  authStore.capabilities = { ...defaultCapabilities };
  authStore.loading = false;
  authStore.error = "";
}

export function setWorkspaceToken(workspaceId: string, token: string): void {
  authStore.tokensByWorkspaceId = {
    ...authStore.tokensByWorkspaceId,
    [workspaceId]: token
  };
}

export function getWorkspaceToken(workspaceId: string): string {
  return authStore.tokensByWorkspaceId[workspaceId] ?? "";
}

export function canAccessAdmin(): boolean {
  return authStore.capabilities.admin_console;
}

export async function refreshMeForCurrentWorkspace(): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    workspaceStore.connectionState = "error";
    authStore.error = "No workspace selected";
    return;
  }

  authStore.loading = true;
  authStore.error = "";

  try {
    if (workspace.mode === "local") {
      const me = await getControlClient().get<Me>("/v1/me");
      applyMe(me);
      workspaceStore.connectionState = "ready";
      return;
    }

    const token = getWorkspaceToken(workspace.id);
    if (token === "") {
      authStore.me = null;
      authStore.capabilities = { ...defaultCapabilities };
      workspaceStore.connectionState = "auth_required";
      return;
    }

    if (!workspace.hub_url) {
      workspaceStore.connectionState = "error";
      authStore.error = "Remote workspace is missing hub_url";
      return;
    }

    const me = await getTargetClient(workspace.hub_url).get<Me>("/v1/me", { token });
    applyMe(me);
    workspaceStore.connectionState = "ready";
  } catch (error) {
    if (workspace.mode === "local") {
      applyMe({
        user_id: "local_user",
        display_name: "local-user",
        workspace_id: workspace.id,
        role: "admin",
        capabilities: {
          admin_console: true,
          resource_write: true,
          execution_control: true
        }
      });
      workspaceStore.connectionState = "ready";
      return;
    }

    authStore.me = null;
    authStore.capabilities = { ...defaultCapabilities };

    const message = formatAuthError(error);
    authStore.error = message;

    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
      workspaceStore.connectionState = "auth_required";
    } else {
      workspaceStore.connectionState = "error";
    }
  } finally {
    authStore.loading = false;
  }
}

function applyMe(me: Me): void {
  authStore.me = me;
  authStore.capabilities = me.capabilities;
  authStore.error = "";
}

function formatAuthError(error: unknown): string {
  if (error instanceof ApiError) {
    return `${error.message} (trace_id: ${error.traceId})`;
  }

  if (error instanceof Error) {
    return error.message;
  }

  return "Unknown auth error";
}
