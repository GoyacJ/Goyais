import { defineStore } from "pinia";

import { pinia } from "@/shared/stores/pinia";
import type {
  McpConnectResult,
  ModelCatalogResponse,
  ModelTestResult,
  ResourceConfig,
  ResourceType,
  WorkspaceProjectConfigItem
} from "@/shared/types/api";

export type EnabledFilter = "all" | "enabled" | "disabled";

export type CursorPageState = {
  limit: number;
  currentCursor: string | null;
  backStack: Array<string | null>;
  nextCursor: string | null;
  loading: boolean;
};

export type ResourceListState = {
  items: ResourceConfig[];
  page: CursorPageState;
  q: string;
  enabledFilter: EnabledFilter;
  loading: boolean;
};

type ResourceStoreState = {
  models: ResourceListState;
  rules: ResourceListState;
  skills: ResourceListState;
  mcps: ResourceListState;
  modelTestResultsByConfigId: Record<string, ModelTestResult>;
  mcpConnectResultsByConfigId: Record<string, McpConnectResult>;
  catalog: ModelCatalogResponse | null;
  catalogRoot: string;
  mcpExport: Record<string, unknown> | null;
  projectBindings: WorkspaceProjectConfigItem[];
  loading: boolean;
  catalogLoading: boolean;
  mcpExportLoading: boolean;
  projectBindingsLoading: boolean;
  error: string;
};

const initialState: ResourceStoreState = {
  models: createResourceListState(),
  rules: createResourceListState(),
  skills: createResourceListState(),
  mcps: createResourceListState(),
  modelTestResultsByConfigId: {},
  mcpConnectResultsByConfigId: {},
  catalog: null,
  catalogRoot: "",
  mcpExport: null,
  projectBindings: [],
  loading: false,
  catalogLoading: false,
  mcpExportLoading: false,
  projectBindingsLoading: false,
  error: ""
};

const useResourceStoreDefinition = defineStore("resource", {
  state: (): ResourceStoreState => ({ ...initialState })
});

export const useResourceStore = useResourceStoreDefinition;
export const resourceStore = useResourceStoreDefinition(pinia);

export function resetResourceStore(): void {
  resourceStore.models = createResourceListState();
  resourceStore.rules = createResourceListState();
  resourceStore.skills = createResourceListState();
  resourceStore.mcps = createResourceListState();
  resourceStore.modelTestResultsByConfigId = {};
  resourceStore.mcpConnectResultsByConfigId = {};
  resourceStore.catalog = null;
  resourceStore.catalogRoot = "";
  resourceStore.mcpExport = null;
  resourceStore.projectBindings = [];
  resourceStore.loading = false;
  resourceStore.catalogLoading = false;
  resourceStore.mcpExportLoading = false;
  resourceStore.projectBindingsLoading = false;
  resourceStore.error = "";
}

export function pickResourceListState(type: ResourceType): ResourceListState {
  if (type === "model") {
    return resourceStore.models;
  }
  if (type === "rule") {
    return resourceStore.rules;
  }
  if (type === "skill") {
    return resourceStore.skills;
  }
  return resourceStore.mcps;
}

export function setResourceSearch(type: ResourceType, query: string): void {
  pickResourceListState(type).q = query;
}

export function setResourceEnabledFilter(type: ResourceType, value: EnabledFilter): void {
  pickResourceListState(type).enabledFilter = value;
}

export function toEnabledQuery(input: EnabledFilter): boolean | undefined {
  if (input === "enabled") {
    return true;
  }
  if (input === "disabled") {
    return false;
  }
  return undefined;
}

function createResourceListState(limit = 50): ResourceListState {
  return {
    items: [],
    page: {
      limit,
      currentCursor: null,
      backStack: [],
      nextCursor: null,
      loading: false
    },
    q: "",
    enabledFilter: "all",
    loading: false
  };
}
