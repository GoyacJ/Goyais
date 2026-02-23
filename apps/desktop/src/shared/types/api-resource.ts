import type { ModelVendorName, ResourceScope, ResourceType, ShareStatus } from "@/shared/types/api-common";

export type Resource = {
  id: string;
  workspace_id: string;
  type: ResourceType;
  name: string;
  source: "workspace_native" | "local_import";
  scope: ResourceScope;
  share_status: ShareStatus;
  owner_user_id: string;
  enabled: boolean;
  description?: string;
  created_at: string;
  updated_at: string;
};

export type ModelVendor = {
  workspace_id: string;
  name: ModelVendorName;
  enabled: boolean;
  updated_at: string;
};

export type ModelCatalogItem = {
  workspace_id: string;
  vendor: ModelVendorName;
  model_id: string;
  enabled: boolean;
  status: "active" | "deprecated" | "preview";
  synced_at: string;
};

export type ResourceImportRequest = {
  resource_type: ResourceType;
  source_id: string;
  target_workspace_id: string;
};

export type ShareRequest = {
  id: string;
  workspace_id: string;
  resource_id: string;
  status: ShareStatus;
  requester_user_id: string;
  approver_user_id?: string;
  created_at: string;
  updated_at: string;
};
