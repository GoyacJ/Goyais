import { getControlClient } from "@/shared/services/clients";
import { withApiFallback } from "@/shared/services/fallback";
import { createMockId, mockData } from "@/shared/services/mockData";
import type {
  ListEnvelope,
  ModelCatalogItem,
  ModelVendorName,
  PaginationQuery,
  Resource,
  ResourceImportRequest,
  ShareRequest,
  ShareStatus
} from "@/shared/types/api";

export async function listResources(workspaceId: string, query: PaginationQuery = {}): Promise<ListEnvelope<Resource>> {
  const search = buildPaginationSearch({ ...query, workspace_id: workspaceId });
  return withApiFallback(
    "resource.list",
    () => getControlClient().get<ListEnvelope<Resource>>(`/v1/resources${search}`),
    () => paginateMock(mockData.resources.filter((resource) => resource.workspace_id === workspaceId), query)
  );
}

export async function importResource(input: ResourceImportRequest): Promise<Resource> {
  return withApiFallback(
    "resource.import",
    () =>
      getControlClient().post<Resource>(`/v1/workspaces/${input.target_workspace_id}/resource-imports`, {
        resource_type: input.resource_type,
        source_id: input.source_id
      }),
    () => {
      const now = new Date().toISOString();
      const created: Resource = {
        id: createMockId("res"),
        workspace_id: input.target_workspace_id,
        type: input.resource_type,
        name: `Imported ${input.resource_type.toUpperCase()}`,
        source: "local_import",
        scope: "private",
        share_status: "pending",
        owner_user_id: "local_user",
        enabled: true,
        created_at: now,
        updated_at: now
      };
      mockData.resources.push(created);
      return created;
    }
  );
}

export async function createShareRequest(workspaceId: string, resourceId: string): Promise<ShareRequest> {
  return withApiFallback(
    "resource.shareRequest",
    () => getControlClient().post<ShareRequest>(`/v1/workspaces/${workspaceId}/share-requests`, { resource_id: resourceId }),
    () => {
      const now = new Date().toISOString();
      const request: ShareRequest = {
        id: createMockId("share"),
        workspace_id: workspaceId,
        resource_id: resourceId,
        status: "pending",
        requester_user_id: "u_dev",
        created_at: now,
        updated_at: now
      };
      mockData.shareRequests.push(request);
      return request;
    }
  );
}

export async function updateShareStatus(requestId: string, status: Extract<ShareStatus, "approved" | "denied" | "revoked">): Promise<void> {
  return withApiFallback(
    "resource.updateShareStatus",
    async () => {
      const endpoint = status === "approved" ? "approve" : status === "denied" ? "reject" : "revoke";
      await getControlClient().post<void>(`/v1/share-requests/${requestId}/${endpoint}`);
    },
    () => {
      const target = mockData.shareRequests.find((request) => request.id === requestId);
      if (target) {
        target.status = status;
        target.updated_at = new Date().toISOString();
      }
    }
  );
}

export async function listModelCatalog(workspaceId: string): Promise<ModelCatalogItem[]> {
  return withApiFallback(
    "resource.modelCatalog",
    () => getControlClient().get<ModelCatalogItem[]>(`/v1/workspaces/${workspaceId}/model-catalog`),
    () => {
      const now = new Date().toISOString();
      return [
        { workspace_id: workspaceId, vendor: "OpenAI", model_id: "gpt-4.1", enabled: true, status: "active", synced_at: now },
        { workspace_id: workspaceId, vendor: "Google", model_id: "gemini-2.0-flash", enabled: true, status: "active", synced_at: now },
        { workspace_id: workspaceId, vendor: "Qwen", model_id: "qwen-max", enabled: true, status: "active", synced_at: now },
        { workspace_id: workspaceId, vendor: "Doubao", model_id: "doubao-pro", enabled: true, status: "preview", synced_at: now },
        { workspace_id: workspaceId, vendor: "Zhipu", model_id: "glm-4.6", enabled: true, status: "active", synced_at: now },
        { workspace_id: workspaceId, vendor: "MiniMax", model_id: "abab6.5-chat", enabled: false, status: "deprecated", synced_at: now },
        { workspace_id: workspaceId, vendor: "Local", model_id: "llama3.1:8b", enabled: true, status: "active", synced_at: now }
      ];
    }
  );
}

export async function syncModelCatalog(workspaceId: string, vendors: ModelVendorName[]): Promise<ModelCatalogItem[]> {
  return withApiFallback(
    "resource.modelCatalogSync",
    () =>
      getControlClient().post<ModelCatalogItem[]>(`/v1/workspaces/${workspaceId}/model-catalog`, {
        vendors
      }),
    () => {
      const now = new Date().toISOString();
      return vendors.map((vendor) => ({
        workspace_id: workspaceId,
        vendor,
        model_id: `${vendor.toLowerCase()}-latest`,
        enabled: true,
        status: "active" as const,
        synced_at: now
      }));
    }
  );
}

function buildPaginationSearch(query: PaginationQuery & { workspace_id?: string }): string {
  const params = new URLSearchParams();
  if (query.workspace_id) {
    params.set("workspace_id", query.workspace_id);
  }
  if (query.cursor) {
    params.set("cursor", query.cursor);
  }
  if (query.limit !== undefined) {
    params.set("limit", String(query.limit));
  }
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
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
