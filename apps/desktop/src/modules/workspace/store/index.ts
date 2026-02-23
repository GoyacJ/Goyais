import { resetAdminStore, refreshAdminData } from "@/modules/admin/store";
import { resetConversationStore } from "@/modules/conversation/store";
import { refreshProjects, resetProjectStore } from "@/modules/project/store";
import { refreshModelCatalog, refreshResources, resetResourceStore } from "@/modules/resource/store";
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
  if (workspaceId === "" || workspaceId === workspaceStore.currentWorkspaceId) {
    return;
  }

  workspaceStore.connectionState = "loading";
  setCurrentWorkspace(workspaceId);
  invalidateWorkspaceScopedState();
  await refreshMeForCurrentWorkspace();
  refreshNavigationVisibility();

  if (!isWorkspaceConnectionReady()) {
    return;
  }

  await reloadWorkspaceScopedData();
}

function invalidateWorkspaceScopedState(): void {
  resetProjectStore();
  resetConversationStore();
  resetResourceStore();
  resetAdminStore();
}

async function reloadWorkspaceScopedData(): Promise<void> {
  const tasks: Array<Promise<unknown>> = [refreshProjects(), refreshResources(), refreshModelCatalog()];
  if (workspaceStore.mode === "remote") {
    tasks.push(refreshAdminData());
  }
  await Promise.all(tasks);
}

function isWorkspaceConnectionReady(): boolean {
  return workspaceStore.connectionState === "ready";
}
