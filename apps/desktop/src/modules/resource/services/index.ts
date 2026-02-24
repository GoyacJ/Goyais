import { getControlClient } from "@/shared/services/clients";
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
  ResourceType,
  WorkspaceAgentConfig
} from "@/shared/types/api";

type ResourceConfigQuery = PaginationQuery & {
  type?: ResourceType;
  q?: string;
  enabled?: boolean;
};

export async function getModelCatalog(workspaceId: string): Promise<ModelCatalogResponse> {
  return getControlClient().get<ModelCatalogResponse>(`/v1/workspaces/${workspaceId}/model-catalog`);
}

export async function reloadModelCatalog(workspaceId: string, source: "manual" | "page_open" | "scheduled" = "manual"): Promise<ModelCatalogResponse> {
  return getControlClient().post<ModelCatalogResponse>(`/v1/workspaces/${workspaceId}/model-catalog`, { source });
}

export async function getCatalogRoot(workspaceId: string): Promise<CatalogRootResponse> {
  return getControlClient().get<CatalogRootResponse>(`/v1/workspaces/${workspaceId}/catalog-root`);
}

export async function updateCatalogRoot(workspaceId: string, catalogRoot: string): Promise<CatalogRootResponse> {
  return getControlClient().request<CatalogRootResponse>(`/v1/workspaces/${workspaceId}/catalog-root`, { method: "PUT", body: { catalog_root: catalogRoot } });
}

export async function listResourceConfigs(workspaceId: string, query: ResourceConfigQuery = {}): Promise<ListEnvelope<ResourceConfig>> {
  const search = buildResourceConfigSearch(query);
  return getControlClient().get<ListEnvelope<ResourceConfig>>(`/v1/workspaces/${workspaceId}/resource-configs${search}`);
}

export async function createResourceConfig(workspaceId: string, input: ResourceConfigCreateRequest): Promise<ResourceConfig> {
  return getControlClient().post<ResourceConfig>(`/v1/workspaces/${workspaceId}/resource-configs`, input);
}

export async function patchResourceConfig(
  workspaceId: string,
  configId: string,
  patch: ResourceConfigPatchRequest
): Promise<ResourceConfig> {
  return getControlClient().request<ResourceConfig>(`/v1/workspaces/${workspaceId}/resource-configs/${configId}`, {
    method: "PATCH",
    body: patch
  });
}

export async function deleteResourceConfig(workspaceId: string, configId: string): Promise<void> {
  await getControlClient().request<void>(`/v1/workspaces/${workspaceId}/resource-configs/${configId}`, { method: "DELETE" });
}

export async function testModelResourceConfig(workspaceId: string, configId: string): Promise<ModelTestResult> {
  return getControlClient().post<ModelTestResult>(`/v1/workspaces/${workspaceId}/resource-configs/${configId}/test`, {});
}

export async function connectMcpResourceConfig(workspaceId: string, configId: string): Promise<McpConnectResult> {
  return getControlClient().post<McpConnectResult>(`/v1/workspaces/${workspaceId}/resource-configs/${configId}/connect`, {});
}

export async function exportMcpConfigs(workspaceId: string): Promise<Record<string, unknown>> {
  return getControlClient().get<Record<string, unknown>>(`/v1/workspaces/${workspaceId}/mcps/export`);
}

export async function getWorkspaceAgentConfig(workspaceId: string): Promise<WorkspaceAgentConfig> {
  return getControlClient().get<WorkspaceAgentConfig>(`/v1/workspaces/${workspaceId}/agent-config`);
}

export async function updateWorkspaceAgentConfig(
  workspaceId: string,
  input: WorkspaceAgentConfig
): Promise<WorkspaceAgentConfig> {
  return getControlClient().request<WorkspaceAgentConfig>(`/v1/workspaces/${workspaceId}/agent-config`, {
    method: "PUT",
    body: input
  });
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
