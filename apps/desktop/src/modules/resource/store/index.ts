import { listWorkspaceProjectConfigs } from "@/modules/project/services";
import {
  connectMcpResourceConfig,
  createResourceConfig,
  deleteResourceConfig,
  exportMcpConfigs,
  getModelCatalog,
  listResourceConfigs,
  patchResourceConfig,
  reloadModelCatalog,
  testModelResourceConfig,
  updateCatalogRoot
} from "@/modules/resource/services";
import {
  pickResourceListState,
  resetResourceStore,
  resourceStore,
  setResourceEnabledFilter,
  setResourceSearch,
  toEnabledQuery,
  type EnabledFilter
} from "@/modules/resource/store/state";
import { toDisplayError } from "@/shared/services/errorMapper";
import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";
import type { McpConnectResult, ModelTestResult, ResourceConfig, ResourceConfigCreateRequest, ResourceConfigPatchRequest, ResourceType } from "@/shared/types/api";

export { resourceStore, resetResourceStore, setResourceSearch, setResourceEnabledFilter, type EnabledFilter };

export async function refreshResources(): Promise<void> {
  await Promise.all([
    refreshResourceConfigsByType("model"),
    refreshResourceConfigsByType("rule"),
    refreshResourceConfigsByType("skill"),
    refreshResourceConfigsByType("mcp"),
    refreshWorkspaceProjectBindings()
  ]);
}

export async function refreshResourceConfigsByType(type: ResourceType): Promise<void> {
  await loadResourceConfigsPage(type, { cursor: null, backStack: [] });
}

export async function loadNextResourceConfigsPage(type: ResourceType): Promise<void> {
  const state = pickResourceListState(type);
  const nextCursor = state.page.nextCursor;
  if (!nextCursor || state.page.loading) {
    return;
  }

  await loadResourceConfigsPage(type, {
    cursor: nextCursor,
    backStack: [...state.page.backStack, state.page.currentCursor]
  });
}

export async function loadPreviousResourceConfigsPage(type: ResourceType): Promise<void> {
  const state = pickResourceListState(type);
  if (state.page.backStack.length === 0 || state.page.loading) {
    return;
  }

  const backStack = [...state.page.backStack];
  const previousCursor = backStack.pop() ?? null;
  await loadResourceConfigsPage(type, { cursor: previousCursor, backStack });
}

async function loadResourceConfigsPage(inputType: ResourceType, input: { cursor: string | null; backStack: Array<string | null> }): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  const state = pickResourceListState(inputType);
  resourceStore.loading = true;
  state.loading = true;
  state.page.loading = true;
  resourceStore.error = "";

  try {
    const response = await listResourceConfigs(workspace.id, {
      cursor: input.cursor ?? undefined,
      limit: state.page.limit,
      type: inputType,
      q: state.q.trim() || undefined,
      enabled: toEnabledQuery(state.enabledFilter)
    });

    state.items = response.items;
    state.page.currentCursor = input.cursor;
    state.page.backStack = input.backStack;
    state.page.nextCursor = response.next_cursor;
  } catch (error) {
    resourceStore.error = toDisplayError(error);
  } finally {
    resourceStore.loading = false;
    state.loading = false;
    state.page.loading = false;
  }
}

export async function refreshModelCatalog(): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  resourceStore.catalogLoading = true;
  resourceStore.error = "";
  try {
    resourceStore.catalog = await getModelCatalog(workspace.id);
  } catch (error) {
    resourceStore.error = toDisplayError(error);
  } finally {
    resourceStore.catalogLoading = false;
  }
}

export async function reloadWorkspaceModelCatalog(): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  resourceStore.catalogLoading = true;
  resourceStore.error = "";
  try {
    resourceStore.catalog = await reloadModelCatalog(workspace.id);
  } catch (error) {
    resourceStore.error = toDisplayError(error);
  } finally {
    resourceStore.catalogLoading = false;
  }
}

export async function syncCatalogRoot(catalogRoot: string): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace || catalogRoot.trim() === "") {
    return;
  }

  try {
    const response = await updateCatalogRoot(workspace.id, catalogRoot.trim());
    resourceStore.catalogRoot = response.catalog_root;
  } catch (error) {
    resourceStore.error = toDisplayError(error);
  }
}

export async function createWorkspaceResourceConfig(input: ResourceConfigCreateRequest): Promise<ResourceConfig | null> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return null;
  }

  try {
    const created = await createResourceConfig(workspace.id, input);
    await refreshResourceConfigsByType(input.type);
    return created;
  } catch (error) {
    resourceStore.error = toDisplayError(error);
    return null;
  }
}

export async function patchWorkspaceResourceConfig(type: ResourceType, configId: string, patch: ResourceConfigPatchRequest): Promise<ResourceConfig | null> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return null;
  }

  try {
    const updated = await patchResourceConfig(workspace.id, configId, patch);
    await refreshResourceConfigsByType(type);
    return updated;
  } catch (error) {
    resourceStore.error = toDisplayError(error);
    return null;
  }
}

export async function deleteWorkspaceResourceConfig(type: ResourceType, configId: string): Promise<boolean> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return false;
  }

  try {
    await deleteResourceConfig(workspace.id, configId);
    if (type === "model") {
      delete resourceStore.modelTestResultsByConfigId[configId];
    }
    if (type === "mcp") {
      delete resourceStore.mcpConnectResultsByConfigId[configId];
    }
    await refreshResourceConfigsByType(type);
    return true;
  } catch (error) {
    resourceStore.error = toDisplayError(error);
    return false;
  }
}

export async function testWorkspaceModelConfig(configId: string): Promise<ModelTestResult | null> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return null;
  }

  try {
    const result = await testModelResourceConfig(workspace.id, configId);
    resourceStore.modelTestResultsByConfigId = {
      ...resourceStore.modelTestResultsByConfigId,
      [configId]: result
    };
    return result;
  } catch (error) {
    resourceStore.error = toDisplayError(error);
    return null;
  }
}

export async function connectWorkspaceMcpConfig(configId: string): Promise<McpConnectResult | null> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return null;
  }

  try {
    const result = await connectMcpResourceConfig(workspace.id, configId);
    resourceStore.mcpConnectResultsByConfigId = {
      ...resourceStore.mcpConnectResultsByConfigId,
      [configId]: result
    };
    await refreshResourceConfigsByType("mcp");
    return result;
  } catch (error) {
    resourceStore.error = toDisplayError(error);
    return null;
  }
}

export async function refreshWorkspaceMcpExport(): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  resourceStore.mcpExportLoading = true;
  try {
    resourceStore.mcpExport = await exportMcpConfigs(workspace.id);
  } catch (error) {
    resourceStore.error = toDisplayError(error);
  } finally {
    resourceStore.mcpExportLoading = false;
  }
}

export async function refreshWorkspaceProjectBindings(): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  resourceStore.projectBindingsLoading = true;
  try {
    resourceStore.projectBindings = await listWorkspaceProjectConfigs(workspace.id);
  } catch (error) {
    resourceStore.error = toDisplayError(error);
  } finally {
    resourceStore.projectBindingsLoading = false;
  }
}
