import { reactive } from "vue";

import type { Workspace, WorkspaceConnection, WorkspaceMode } from "@/shared/types/api";

export type ConnectionState = "idle" | "loading" | "auth_required" | "ready" | "error";

const CURRENT_WORKSPACE_STORAGE_KEY = "goyais.workspace.current";

type WorkspaceState = {
  workspaces: Workspace[];
  connectionsByWorkspaceId: Record<string, WorkspaceConnection>;
  currentWorkspaceId: string;
  mode: WorkspaceMode;
  connectionState: ConnectionState;
  loading: boolean;
  error: string;
};

const initialState: WorkspaceState = {
  workspaces: [],
  connectionsByWorkspaceId: {},
  currentWorkspaceId: "",
  mode: "local",
  connectionState: "idle",
  loading: false,
  error: ""
};

export const workspaceStore = reactive<WorkspaceState>({ ...initialState });

export function resetWorkspaceStore(): void {
  workspaceStore.workspaces = [];
  workspaceStore.connectionsByWorkspaceId = {};
  workspaceStore.currentWorkspaceId = "";
  workspaceStore.mode = "local";
  workspaceStore.connectionState = "idle";
  workspaceStore.loading = false;
  workspaceStore.error = "";
  clearPersistedWorkspaceId();
}

export function setWorkspaces(workspaces: Workspace[]): void {
  workspaceStore.workspaces = [...workspaces];

  const hasCurrentWorkspace = workspaceStore.workspaces.some((workspace) => workspace.id === workspaceStore.currentWorkspaceId);
  if (!hasCurrentWorkspace) {
    workspaceStore.currentWorkspaceId = "";
  }

  if (workspaceStore.currentWorkspaceId === "") {
    const persistedWorkspaceId = readPersistedWorkspaceId();
    const persistedWorkspace = workspaceStore.workspaces.find((workspace) => workspace.id === persistedWorkspaceId);
    const defaultWorkspace =
      persistedWorkspace ??
      workspaceStore.workspaces.find((workspace) => workspace.is_default_local) ??
      workspaceStore.workspaces[0];
    if (defaultWorkspace) {
      workspaceStore.currentWorkspaceId = defaultWorkspace.id;
    }
  }

  syncModeWithCurrentWorkspace();
  persistWorkspaceId(workspaceStore.currentWorkspaceId);
}

export function upsertWorkspace(workspace: Workspace): void {
  const index = workspaceStore.workspaces.findIndex((item) => item.id === workspace.id);
  if (index >= 0) {
    workspaceStore.workspaces[index] = workspace;
  } else {
    workspaceStore.workspaces.push(workspace);
  }
}

export function setWorkspaceConnection(connection: WorkspaceConnection): void {
  workspaceStore.connectionsByWorkspaceId = {
    ...workspaceStore.connectionsByWorkspaceId,
    [connection.workspace_id]: connection
  };
}

export function getWorkspaceConnection(workspaceId: string): WorkspaceConnection | undefined {
  return workspaceStore.connectionsByWorkspaceId[workspaceId];
}

export function setCurrentWorkspace(workspaceId: string): void {
  workspaceStore.currentWorkspaceId = workspaceId;
  syncModeWithCurrentWorkspace();
  persistWorkspaceId(workspaceStore.currentWorkspaceId);
}

export function getCurrentWorkspace(): Workspace | undefined {
  return workspaceStore.workspaces.find((workspace) => workspace.id === workspaceStore.currentWorkspaceId);
}

function syncModeWithCurrentWorkspace(): void {
  const current = getCurrentWorkspace();
  workspaceStore.mode = current?.mode ?? "local";
}

function readPersistedWorkspaceId(): string {
  if (typeof window === "undefined") {
    return "";
  }

  try {
    return window.localStorage.getItem(CURRENT_WORKSPACE_STORAGE_KEY) ?? "";
  } catch {
    return "";
  }
}

function persistWorkspaceId(workspaceId: string): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    if (workspaceId === "") {
      window.localStorage.removeItem(CURRENT_WORKSPACE_STORAGE_KEY);
      return;
    }

    window.localStorage.setItem(CURRENT_WORKSPACE_STORAGE_KEY, workspaceId);
  } catch {
    // ignore localStorage failures to avoid blocking workspace switch
  }
}

function clearPersistedWorkspaceId(): void {
  persistWorkspaceId("");
}
