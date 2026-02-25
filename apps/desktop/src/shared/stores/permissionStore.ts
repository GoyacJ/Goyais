import { defineStore } from "pinia";

import { pinia } from "@/shared/stores/pinia";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import type { PermissionSnapshot } from "@/shared/types/api";

type PermissionState = {
  snapshotsByWorkspaceId: Record<string, PermissionSnapshot>;
};

const usePermissionStoreDefinition = defineStore("permission", {
  state: (): PermissionState => ({
    snapshotsByWorkspaceId: {}
  })
});

export const usePermissionStore = usePermissionStoreDefinition;
export const permissionStore = usePermissionStoreDefinition(pinia);

export function resetPermissionStore(): void {
  permissionStore.snapshotsByWorkspaceId = {};
}

export function setWorkspacePermissionSnapshot(workspaceId: string, snapshot: PermissionSnapshot): void {
  permissionStore.snapshotsByWorkspaceId = {
    ...permissionStore.snapshotsByWorkspaceId,
    [workspaceId]: snapshot
  };
}

export function clearWorkspacePermissionSnapshot(workspaceId: string): void {
  if (workspaceId === "") {
    return;
  }
  const next = { ...permissionStore.snapshotsByWorkspaceId };
  delete next[workspaceId];
  permissionStore.snapshotsByWorkspaceId = next;
}

export function getWorkspacePermissionSnapshot(workspaceId: string): PermissionSnapshot | null {
  if (workspaceId === "") {
    return null;
  }
  return permissionStore.snapshotsByWorkspaceId[workspaceId] ?? null;
}

export function getCurrentPermissionSnapshot(): PermissionSnapshot | null {
  return getWorkspacePermissionSnapshot(workspaceStore.currentWorkspaceId);
}
