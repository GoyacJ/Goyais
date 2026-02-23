import { reactive } from "vue";

import {
  createShareRequest,
  importResource,
  listResources,
  updateShareStatus
} from "@/modules/resource/services";
import { toDisplayError } from "@/shared/services/errorMapper";
import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";
import type { Resource, ResourceImportRequest, ShareStatus } from "@/shared/types/api";

type ResourceState = {
  items: Resource[];
  loading: boolean;
  error: string;
};

const initialState: ResourceState = {
  items: [],
  loading: false,
  error: ""
};

export const resourceStore = reactive<ResourceState>({ ...initialState });

export function resetResourceStore(): void {
  resourceStore.items = [];
  resourceStore.loading = false;
  resourceStore.error = "";
}

export async function refreshResources(): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  resourceStore.loading = true;
  resourceStore.error = "";

  try {
    const response = await listResources(workspace.id);
    resourceStore.items = response.items;
  } catch (error) {
    resourceStore.error = toDisplayError(error);
  } finally {
    resourceStore.loading = false;
  }
}

export async function importWorkspaceResource(input: Omit<ResourceImportRequest, "target_workspace_id">): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  try {
    const created = await importResource({
      ...input,
      target_workspace_id: workspace.id
    });
    resourceStore.items.push(created);
  } catch (error) {
    resourceStore.error = toDisplayError(error);
  }
}

export async function requestResourceShare(resourceId: string): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  try {
    await createShareRequest(workspace.id, resourceId);
    await refreshResources();
  } catch (error) {
    resourceStore.error = toDisplayError(error);
  }
}

export async function changeResourceShareStatus(
  requestId: string,
  status: Extract<ShareStatus, "approved" | "denied" | "revoked">
): Promise<void> {
  try {
    await updateShareStatus(requestId, status);
    await refreshResources();
  } catch (error) {
    resourceStore.error = toDisplayError(error);
  }
}
