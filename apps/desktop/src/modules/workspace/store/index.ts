import { listWorkspaces } from "@/modules/workspace/services";
import {
  initializeGeneralSettings,
  resetGeneralSettings,
  resetGeneralSettingsStoreForTest,
  updateGeneralSetting,
  useGeneralSettings
} from "@/modules/workspace/store/generalSettingsStore";
import { toDisplayError } from "@/shared/services/errorMapper";
import { refreshMeForCurrentWorkspace } from "@/shared/stores/authStore";
import { refreshNavigationVisibility } from "@/shared/stores/navigationStore";
import {
  getCurrentWorkspace,
  getWorkspaceConnection,
  resetWorkspaceStore,
  setCurrentWorkspace,
  setWorkspaceConnection,
  setWorkspaces,
  upsertWorkspace,
  workspaceStore
} from "@/shared/stores/workspaceStore";

export {
  getCurrentWorkspace,
  initializeGeneralSettings,
  getWorkspaceConnection,
  resetWorkspaceStore,
  resetGeneralSettings,
  resetGeneralSettingsStoreForTest,
  setCurrentWorkspace,
  setWorkspaceConnection,
  setWorkspaces,
  updateGeneralSetting,
  useGeneralSettings,
  upsertWorkspace,
  workspaceStore
};

export async function initializeWorkspaceContext(): Promise<void> {
  workspaceStore.loading = true;
  workspaceStore.error = "";

  try {
    const response = await listWorkspaces();
    setWorkspaces(response.items);

    if (workspaceStore.currentWorkspaceId !== "") {
      await refreshMeForCurrentWorkspace();
    }

    refreshNavigationVisibility();
  } catch (error) {
    workspaceStore.error = toDisplayError(error);
    workspaceStore.connectionState = "error";
  } finally {
    workspaceStore.loading = false;
  }
}

export async function switchWorkspaceContext(workspaceId: string): Promise<void> {
  if (workspaceId === "") {
    return;
  }

  setCurrentWorkspace(workspaceId);
  await refreshMeForCurrentWorkspace();
  refreshNavigationVisibility();
}
