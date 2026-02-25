import { defineStore } from "pinia";
import { useStorage } from "@vueuse/core";

import { pinia } from "@/shared/stores/pinia";
import type { Workspace, WorkspaceConnection, WorkspaceMode } from "@/shared/types/api";

export type ConnectionState = "idle" | "loading" | "auth_required" | "ready" | "error";

const CURRENT_WORKSPACE_STORAGE_KEY = "goyais.workspace.current";
const RECENT_WORKSPACE_ORDER_STORAGE_KEY = "goyais.workspace.recent_order";
const persistedWorkspaceId = useStorage<string>(CURRENT_WORKSPACE_STORAGE_KEY, "", undefined, { flush: "sync" });
const persistedWorkspaceRecentOrder = useStorage<string[]>(RECENT_WORKSPACE_ORDER_STORAGE_KEY, [], undefined, { flush: "sync" });

type WorkspaceState = {
  workspaces: Workspace[];
  connectionsByWorkspaceId: Record<string, WorkspaceConnection>;
  currentWorkspaceId: string;
  recentWorkspaceOrder: string[];
  mode: WorkspaceMode;
  connectionState: ConnectionState;
  loading: boolean;
  error: string;
};

const initialState: WorkspaceState = {
  workspaces: [],
  connectionsByWorkspaceId: {},
  currentWorkspaceId: "",
  recentWorkspaceOrder: [],
  mode: "local",
  connectionState: "idle",
  loading: false,
  error: ""
};

const useWorkspaceStoreDefinition = defineStore("workspace", {
  state: (): WorkspaceState => ({
    ...initialState,
    recentWorkspaceOrder: readPersistedWorkspaceRecentOrder()
  })
});

export const useWorkspaceStore = useWorkspaceStoreDefinition;
export const workspaceStore = useWorkspaceStoreDefinition(pinia);

export function resetWorkspaceStore(): void {
  workspaceStore.workspaces = [];
  workspaceStore.connectionsByWorkspaceId = {};
  workspaceStore.currentWorkspaceId = "";
  workspaceStore.recentWorkspaceOrder = [];
  workspaceStore.mode = "local";
  workspaceStore.connectionState = "idle";
  workspaceStore.loading = false;
  workspaceStore.error = "";
  clearPersistedWorkspaceId();
  clearPersistedWorkspaceRecentOrder();
}

export function setWorkspaces(workspaces: Workspace[]): void {
  const normalizedWorkspaces = ensureLocalWorkspace(workspaces);
  const normalizedRecentOrder = normalizeRecentOrder(workspaceStore.recentWorkspaceOrder, normalizedWorkspaces);
  workspaceStore.recentWorkspaceOrder = normalizedRecentOrder;
  persistWorkspaceRecentOrder(normalizedRecentOrder);

  workspaceStore.workspaces = sortWorkspacesForSwitcher(normalizedWorkspaces, normalizedRecentOrder);
  ensureCurrentWorkspaceSelection();
  syncModeWithCurrentWorkspace();
  persistWorkspaceId(workspaceStore.currentWorkspaceId);
}

export function upsertWorkspace(workspace: Workspace): void {
  const byId = new Map(workspaceStore.workspaces.map((item) => [item.id, item] as const));
  byId.set(workspace.id, workspace);
  setWorkspaces([...byId.values()]);
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
  if (!workspaceStore.workspaces.some((workspace) => workspace.id === workspaceId)) {
    return;
  }
  workspaceStore.currentWorkspaceId = workspaceId;
  touchWorkspaceRecentOrder(workspaceId);
  syncModeWithCurrentWorkspace();
  persistWorkspaceId(workspaceStore.currentWorkspaceId);
}

export function getCurrentWorkspace(): Workspace | undefined {
  return workspaceStore.workspaces.find((workspace) => workspace.id === workspaceStore.currentWorkspaceId);
}

function ensureCurrentWorkspaceSelection(): void {
  const hasCurrentWorkspace = workspaceStore.workspaces.some((workspace) => workspace.id === workspaceStore.currentWorkspaceId);
  if (!hasCurrentWorkspace) {
    workspaceStore.currentWorkspaceId = "";
  }

  if (workspaceStore.currentWorkspaceId === "") {
    const persistedWorkspaceId = readPersistedWorkspaceId();
    const persistedWorkspace = workspaceStore.workspaces.find((workspace) => workspace.id === persistedWorkspaceId);
    const defaultWorkspace =
      persistedWorkspace ??
      workspaceStore.workspaces.find((workspace) => workspace.is_default_local || workspace.mode === "local") ??
      workspaceStore.workspaces[0];
    if (defaultWorkspace) {
      workspaceStore.currentWorkspaceId = defaultWorkspace.id;
    }
  }
}

function touchWorkspaceRecentOrder(workspaceId: string): void {
  const workspace = workspaceStore.workspaces.find((item) => item.id === workspaceId);
  if (!workspace || workspace.mode !== "remote") {
    return;
  }

  const nextOrder = [workspaceId, ...workspaceStore.recentWorkspaceOrder.filter((item) => item !== workspaceId)];
  workspaceStore.recentWorkspaceOrder = nextOrder;
  persistWorkspaceRecentOrder(nextOrder);
  workspaceStore.workspaces = sortWorkspacesForSwitcher(workspaceStore.workspaces, nextOrder);
}

function syncModeWithCurrentWorkspace(): void {
  const current = getCurrentWorkspace();
  workspaceStore.mode = current?.mode ?? "local";
}

function readPersistedWorkspaceId(): string {
  return typeof persistedWorkspaceId.value === "string" ? persistedWorkspaceId.value : "";
}

function persistWorkspaceId(workspaceId: string): void {
  persistedWorkspaceId.value = workspaceId;
}

function clearPersistedWorkspaceId(): void {
  persistWorkspaceId("");
}

function readPersistedWorkspaceRecentOrder(): string[] {
  const order = persistedWorkspaceRecentOrder.value;
  if (!Array.isArray(order)) {
    return [];
  }
  return order.filter((item): item is string => typeof item === "string" && item.trim() !== "");
}

function persistWorkspaceRecentOrder(order: string[]): void {
  persistedWorkspaceRecentOrder.value = [...order];
}

function clearPersistedWorkspaceRecentOrder(): void {
  persistWorkspaceRecentOrder([]);
}

function ensureLocalWorkspace(workspaces: Workspace[]): Workspace[] {
  const byId = new Map(workspaces.map((workspace) => [workspace.id, workspace] as const));
  let localWorkspace = workspaces.find((workspace) => workspace.mode === "local" || workspace.is_default_local);
  if (!localWorkspace) {
    localWorkspace = {
      id: "ws_local",
      name: "Local Workspace",
      mode: "local",
      hub_url: null,
      is_default_local: true,
      created_at: new Date().toISOString(),
      login_disabled: true,
      auth_mode: "disabled"
    };
  }
  byId.set(localWorkspace.id, localWorkspace);
  return [...byId.values()];
}

function sortWorkspacesForSwitcher(workspaces: Workspace[], recentOrder: string[]): Workspace[] {
  const localWorkspace = workspaces.find((workspace) => workspace.mode === "local" || workspace.is_default_local);
  const remoteWorkspaces = workspaces.filter((workspace) => workspace.mode === "remote");
  const others = workspaces.filter(
    (workspace) =>
      workspace.id !== localWorkspace?.id && workspace.mode !== "remote" && workspace.mode !== "local" && !workspace.is_default_local
  );

  const recentOrderIndex = new Map(recentOrder.map((workspaceId, index) => [workspaceId, index] as const));
  remoteWorkspaces.sort((left, right) => {
    const leftIndex = recentOrderIndex.get(left.id);
    const rightIndex = recentOrderIndex.get(right.id);
    if (leftIndex !== undefined && rightIndex !== undefined && leftIndex !== rightIndex) {
      return leftIndex - rightIndex;
    }
    if (leftIndex !== undefined) {
      return -1;
    }
    if (rightIndex !== undefined) {
      return 1;
    }

    const leftCreatedAt = Date.parse(left.created_at);
    const rightCreatedAt = Date.parse(right.created_at);
    const leftValid = Number.isNaN(leftCreatedAt) === false;
    const rightValid = Number.isNaN(rightCreatedAt) === false;
    if (leftValid && rightValid && leftCreatedAt !== rightCreatedAt) {
      return leftCreatedAt - rightCreatedAt;
    }
    return left.name.localeCompare(right.name);
  });

  const sorted: Workspace[] = [];
  if (localWorkspace) {
    sorted.push(localWorkspace);
  }
  sorted.push(...remoteWorkspaces);
  sorted.push(...others);
  return sorted;
}

function normalizeRecentOrder(order: string[], workspaces: Workspace[]): string[] {
  const remoteIDs = new Set(workspaces.filter((workspace) => workspace.mode === "remote").map((workspace) => workspace.id));
  return order.filter((workspaceId) => remoteIDs.has(workspaceId));
}
