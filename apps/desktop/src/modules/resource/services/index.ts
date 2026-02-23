import { getControlClient } from "@/shared/services/clients";
import { withApiFallback } from "@/shared/services/fallback";
import { createMockId, mockData } from "@/shared/services/mockData";
import type {
  CatalogRootResponse,
  ListEnvelope,
  McpConnectResult,
  ModelCatalogResponse,
  ModelTestResult,
  PaginationQuery,
  ResourceConfig,
  ResourceConfigCreateRequest,
  ResourceConfigPatchRequest,
  ResourceType
} from "@/shared/types/api";

type ResourceConfigQuery = PaginationQuery & {
  type?: ResourceType;
  q?: string;
  enabled?: boolean;
};

export async function getModelCatalog(workspaceId: string): Promise<ModelCatalogResponse> {
  return withApiFallback(
    "resource.catalog.get",
    () => getControlClient().get<ModelCatalogResponse>(`/v1/workspaces/${workspaceId}/model-catalog`),
    () => buildMockCatalog(workspaceId)
  );
}

export async function reloadModelCatalog(workspaceId: string): Promise<ModelCatalogResponse> {
  return withApiFallback(
    "resource.catalog.reload",
    () => getControlClient().post<ModelCatalogResponse>(`/v1/workspaces/${workspaceId}/model-catalog`, {}),
    () => buildMockCatalog(workspaceId)
  );
}

export async function getCatalogRoot(workspaceId: string): Promise<CatalogRootResponse> {
  return withApiFallback(
    "resource.catalogRoot.get",
    () => getControlClient().get<CatalogRootResponse>(`/v1/workspaces/${workspaceId}/catalog-root`),
    () => ({
      workspace_id: workspaceId,
      catalog_root: "~/.goyais",
      updated_at: new Date().toISOString()
    })
  );
}

export async function updateCatalogRoot(workspaceId: string, catalogRoot: string): Promise<CatalogRootResponse> {
  return withApiFallback(
    "resource.catalogRoot.update",
    () => getControlClient().request<CatalogRootResponse>(`/v1/workspaces/${workspaceId}/catalog-root`, { method: "PUT", body: { catalog_root: catalogRoot } }),
    () => ({
      workspace_id: workspaceId,
      catalog_root: catalogRoot,
      updated_at: new Date().toISOString()
    })
  );
}

export async function listResourceConfigs(workspaceId: string, query: ResourceConfigQuery = {}): Promise<ListEnvelope<ResourceConfig>> {
  const search = buildResourceConfigSearch(query);
  return withApiFallback(
    "resource.config.list",
    () => getControlClient().get<ListEnvelope<ResourceConfig>>(`/v1/workspaces/${workspaceId}/resource-configs${search}`),
    () => paginateMock(filterMockConfigs(workspaceId, query), query)
  );
}

export async function createResourceConfig(workspaceId: string, input: ResourceConfigCreateRequest): Promise<ResourceConfig> {
  return withApiFallback(
    "resource.config.create",
    () => getControlClient().post<ResourceConfig>(`/v1/workspaces/${workspaceId}/resource-configs`, input),
    () => {
      const now = new Date().toISOString();
      const created: ResourceConfig = {
        id: createMockId("rc"),
        workspace_id: workspaceId,
        type: input.type,
        name: input.name,
        enabled: input.enabled ?? true,
        model: input.model,
        rule: input.rule,
        skill: input.skill,
        mcp: input.mcp,
        created_at: now,
        updated_at: now
      };
      ensureMockResourceConfigs();
      mockData.resourceConfigs.push(created);
      return created;
    }
  );
}

export async function patchResourceConfig(
  workspaceId: string,
  configId: string,
  patch: ResourceConfigPatchRequest
): Promise<ResourceConfig> {
  return withApiFallback(
    "resource.config.patch",
    () =>
      getControlClient().request<ResourceConfig>(`/v1/workspaces/${workspaceId}/resource-configs/${configId}`, {
        method: "PATCH",
        body: patch
      }),
    () => {
      ensureMockResourceConfigs();
      const target = mockData.resourceConfigs.find((item) => item.id === configId && item.workspace_id === workspaceId);
      if (!target) {
        throw new Error("Resource config not found");
      }
      if (patch.name !== undefined) {
        target.name = patch.name;
      }
      if (patch.enabled !== undefined) {
        target.enabled = patch.enabled;
      }
      if (patch.model !== undefined) {
        target.model = patch.model;
      }
      if (patch.rule !== undefined) {
        target.rule = patch.rule;
      }
      if (patch.skill !== undefined) {
        target.skill = patch.skill;
      }
      if (patch.mcp !== undefined) {
        target.mcp = patch.mcp;
      }
      target.updated_at = new Date().toISOString();
      return target;
    }
  );
}

export async function deleteResourceConfig(workspaceId: string, configId: string): Promise<void> {
  return withApiFallback(
    "resource.config.delete",
    async () => {
      await getControlClient().request<void>(`/v1/workspaces/${workspaceId}/resource-configs/${configId}`, { method: "DELETE" });
    },
    () => {
      ensureMockResourceConfigs();
      mockData.resourceConfigs = mockData.resourceConfigs.filter((item) => !(item.workspace_id === workspaceId && item.id === configId));
    }
  );
}

export async function testModelResourceConfig(workspaceId: string, configId: string): Promise<ModelTestResult> {
  return withApiFallback(
    "resource.config.testModel",
    () => getControlClient().post<ModelTestResult>(`/v1/workspaces/${workspaceId}/resource-configs/${configId}/test`, {}),
    () => ({
      config_id: configId,
      status: "success",
      latency_ms: 1,
      message: "mock model test success",
      tested_at: new Date().toISOString()
    })
  );
}

export async function connectMcpResourceConfig(workspaceId: string, configId: string): Promise<McpConnectResult> {
  return withApiFallback(
    "resource.config.connectMcp",
    () => getControlClient().post<McpConnectResult>(`/v1/workspaces/${workspaceId}/resource-configs/${configId}/connect`, {}),
    () => ({
      config_id: configId,
      status: "connected",
      tools: ["tools.list", "resources.list"],
      message: "mock mcp connected",
      connected_at: new Date().toISOString()
    })
  );
}

export async function exportMcpConfigs(workspaceId: string): Promise<Record<string, unknown>> {
  return withApiFallback(
    "resource.config.exportMcp",
    () => getControlClient().get<Record<string, unknown>>(`/v1/workspaces/${workspaceId}/mcps/export`),
    () => ({
      workspace_id: workspaceId,
      mcps: []
    })
  );
}

function buildResourceConfigSearch(query: ResourceConfigQuery): string {
  const params = new URLSearchParams();
  if (query.cursor) {
    params.set("cursor", query.cursor);
  }
  if (query.limit !== undefined) {
    params.set("limit", String(query.limit));
  }
  if (query.type) {
    params.set("type", query.type);
  }
  if (query.q) {
    params.set("q", query.q);
  }
  if (query.enabled !== undefined) {
    params.set("enabled", query.enabled ? "true" : "false");
  }
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}

function filterMockConfigs(workspaceId: string, query: ResourceConfigQuery): ResourceConfig[] {
  ensureMockResourceConfigs();
  return mockData.resourceConfigs.filter((item) => {
    if (item.workspace_id !== workspaceId) {
      return false;
    }
    if (query.type && item.type !== query.type) {
      return false;
    }
    if (query.enabled !== undefined && item.enabled !== query.enabled) {
      return false;
    }
    if (query.q && item.name.toLowerCase().includes(query.q.toLowerCase()) === false) {
      return false;
    }
    return true;
  });
}

function paginateMock<T>(items: T[], query: PaginationQuery): ListEnvelope<T> {
  const start = Number.parseInt(query.cursor ?? "0", 10);
  const safeStart = Number.isNaN(start) || start < 0 ? 0 : start;
  const limit = query.limit !== undefined && query.limit > 0 ? query.limit : 20;
  const end = Math.min(safeStart + limit, items.length);
  return {
    items: items.slice(safeStart, end),
    next_cursor: end < items.length ? String(end) : null
  };
}

function buildMockCatalog(workspaceId: string): ModelCatalogResponse {
  const now = new Date().toISOString();
  return {
    workspace_id: workspaceId,
    revision: Date.now(),
    updated_at: now,
    source: "~/.goyais/goyais/catalog/models.json",
    vendors: [
      { name: "OpenAI", models: [{ id: "gpt-4.1", label: "GPT-4.1", enabled: true }] },
      { name: "Google", models: [{ id: "gemini-2.0-flash", label: "Gemini 2.0 Flash", enabled: true }] },
      { name: "Qwen", models: [{ id: "qwen-max", label: "Qwen Max", enabled: true }] },
      { name: "Doubao", models: [{ id: "doubao-pro-32k", label: "Doubao Pro 32k", enabled: true }] },
      { name: "Zhipu", models: [{ id: "glm-4-plus", label: "GLM-4-Plus", enabled: true }] },
      { name: "MiniMax", models: [{ id: "MiniMax-Text-01", label: "MiniMax Text 01", enabled: true }] },
      { name: "Local", models: [{ id: "llama3.1:8b", label: "Llama 3.1 8B", enabled: true }] }
    ]
  };
}

function ensureMockResourceConfigs(): void {
  if (Array.isArray(mockData.resourceConfigs)) {
    return;
  }
  mockData.resourceConfigs = [];
}

