import { reactive } from "vue";

import type { Workspace, WorkspaceMode } from "@/shared/types/api";

export type ConnectionState = "idle" | "loading" | "auth_required" | "ready" | "error";

type WorkspaceState = {
  workspaces: Workspace[];
  currentWorkspaceId: string;
  mode: WorkspaceMode;
  connectionState: ConnectionState;
  loading: boolean;
  error: string;
};

const initialState: WorkspaceState = {
  workspaces: [],
  currentWorkspaceId: "",
  mode: "local",
  connectionState: "idle",
  loading: false,
  error: ""
};

export const workspaceStore = reactive<WorkspaceState>({ ...initialState });

export function resetWorkspaceStore(): void {
  workspaceStore.workspaces = [];
  workspaceStore.currentWorkspaceId = "";
  workspaceStore.mode = "local";
  workspaceStore.connectionState = "idle";
  workspaceStore.loading = false;
  workspaceStore.error = "";
}

export function setWorkspaces(workspaces: Workspace[]): void {
  workspaceStore.workspaces = [...workspaces];

  if (workspaceStore.currentWorkspaceId === "") {
    const defaultWorkspace = workspaceStore.workspaces.find((workspace) => workspace.is_default_local) ?? workspaceStore.workspaces[0];
    if (defaultWorkspace) {
      workspaceStore.currentWorkspaceId = defaultWorkspace.id;
    }
  }

  syncModeWithCurrentWorkspace();
}

export function upsertWorkspace(workspace: Workspace): void {
  const index = workspaceStore.workspaces.findIndex((item) => item.id === workspace.id);
  if (index >= 0) {
    workspaceStore.workspaces[index] = workspace;
  } else {
    workspaceStore.workspaces.push(workspace);
  }
}

export function setCurrentWorkspace(workspaceId: string): void {
  workspaceStore.currentWorkspaceId = workspaceId;
  syncModeWithCurrentWorkspace();
}

export function getCurrentWorkspace(): Workspace | undefined {
  return workspaceStore.workspaces.find((workspace) => workspace.id === workspaceStore.currentWorkspaceId);
}

function syncModeWithCurrentWorkspace(): void {
  const current = getCurrentWorkspace();
  workspaceStore.mode = current?.mode ?? "local";
}
