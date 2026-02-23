import { reactive } from "vue";

import {
  createShareRequest,
  importResource,
  listModelCatalog,
  listResources,
  syncModelCatalog,
  updateShareStatus
} from "@/modules/resource/services";
import { toDisplayError } from "@/shared/services/errorMapper";
import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";
import type { ModelCatalogItem, ModelVendorName, Resource, ResourceImportRequest, ShareStatus } from "@/shared/types/api";

type CursorPageState = {
  limit: number;
  currentCursor: string | null;
  backStack: Array<string | null>;
  nextCursor: string | null;
  loading: boolean;
};

type ResourceState = {
  items: Resource[];
  resourcesPage: CursorPageState;
  modelCatalog: ModelCatalogItem[];
  modelCatalogSyncing: boolean;
  loading: boolean;
  error: string;
};

const initialState: ResourceState = {
  items: [],
  resourcesPage: createInitialPageState(),
  modelCatalog: [],
  modelCatalogSyncing: false,
  loading: false,
  error: ""
};

export const resourceStore = reactive<ResourceState>({ ...initialState });

export function resetResourceStore(): void {
  resourceStore.items = [];
  resourceStore.resourcesPage = createInitialPageState();
  resourceStore.modelCatalog = [];
  resourceStore.modelCatalogSyncing = false;
  resourceStore.loading = false;
  resourceStore.error = "";
}

export async function refreshResources(): Promise<void> {
  await loadResourcesPage({ cursor: null, backStack: [] });
}

export async function loadNextResourcesPage(): Promise<void> {
  const nextCursor = resourceStore.resourcesPage.nextCursor;
  if (!nextCursor || resourceStore.resourcesPage.loading) {
    return;
  }
  await loadResourcesPage({
    cursor: nextCursor,
    backStack: [...resourceStore.resourcesPage.backStack, resourceStore.resourcesPage.currentCursor]
  });
}

export async function loadPreviousResourcesPage(): Promise<void> {
  if (resourceStore.resourcesPage.backStack.length === 0 || resourceStore.resourcesPage.loading) {
    return;
  }
  const backStack = [...resourceStore.resourcesPage.backStack];
  const previousCursor = backStack.pop() ?? null;
  await loadResourcesPage({ cursor: previousCursor, backStack });
}

async function loadResourcesPage(input: { cursor: string | null; backStack: Array<string | null> }): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  resourceStore.loading = true;
  resourceStore.resourcesPage.loading = true;
  resourceStore.error = "";

  try {
    const response = await listResources(workspace.id, {
      cursor: input.cursor ?? undefined,
      limit: resourceStore.resourcesPage.limit
    });
    resourceStore.items = response.items;
    resourceStore.resourcesPage.currentCursor = input.cursor;
    resourceStore.resourcesPage.backStack = input.backStack;
    resourceStore.resourcesPage.nextCursor = response.next_cursor;
  } catch (error) {
    resourceStore.error = toDisplayError(error);
  } finally {
    resourceStore.loading = false;
    resourceStore.resourcesPage.loading = false;
  }
}

export async function refreshModelCatalog(): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  try {
    resourceStore.modelCatalog = await listModelCatalog(workspace.id);
  } catch (error) {
    resourceStore.error = toDisplayError(error);
  }
}

export async function syncWorkspaceModelCatalog(vendors: ModelVendorName[]): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  resourceStore.modelCatalogSyncing = true;
  try {
    resourceStore.modelCatalog = await syncModelCatalog(workspace.id, vendors);
  } catch (error) {
    resourceStore.error = toDisplayError(error);
  } finally {
    resourceStore.modelCatalogSyncing = false;
  }
}

export async function importWorkspaceResource(input: Omit<ResourceImportRequest, "target_workspace_id">): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  try {
    await importResource({
      ...input,
      target_workspace_id: workspace.id
    });
    await refreshResources();
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

function createInitialPageState(limit = 20): CursorPageState {
  return {
    limit,
    currentCursor: null,
    backStack: [],
    nextCursor: null,
    loading: false
  };
}
