import type { ModelVendorName, ResourceScope, ResourceType, ShareStatus } from "./api-common";

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

export type ModelCatalogModel = {
  id: string;
  label: string;
  enabled: boolean;
};

export type ModelCatalogVendorAuth = {
  type: "none" | "http_bearer" | "api_key_header";
  header?: string;
  scheme?: string;
  api_key_env?: string;
};

export type ModelCatalogVendor = {
  name: ModelVendorName;
  homepage?: string;
  docs?: string;
  base_url: string;
  base_urls?: Record<string, string>;
  auth: ModelCatalogVendorAuth;
  models: ModelCatalogModel[];
  notes?: string[];
};

export type ModelCatalogResponse = {
  workspace_id: string;
  revision: number;
  updated_at: string;
  source: string;
  vendors: ModelCatalogVendor[];
};

export type CatalogRootResponse = {
  workspace_id: string;
  catalog_root: string;
  updated_at: string;
};

export type ModelSpec = {
  vendor: ModelVendorName;
  model_id: string;
  base_url?: string;
  base_url_key?: string;
  api_key?: string;
  api_key_masked?: string;
  runtime?: {
    request_timeout_ms?: number;
  };
  params?: Record<string, unknown>;
};

export type RuleSpec = {
  content: string;
};

export type SkillSpec = {
  content: string;
};

export type McpSpec = {
  transport: "http_sse" | "stdio";
  endpoint?: string;
  command?: string;
  env?: Record<string, string>;
  status?: string;
  tools?: string[];
  last_error?: string;
  last_connected_at?: string;
};

export type ResourceConfig = {
  id: string;
  workspace_id: string;
  type: ResourceType;
  name?: string;
  enabled: boolean;
  model?: ModelSpec;
  rule?: RuleSpec;
  skill?: SkillSpec;
  mcp?: McpSpec;
  created_at: string;
  updated_at: string;
};

export type ResourceConfigCreateRequest = {
  type: ResourceType;
  name?: string;
  enabled?: boolean;
  model?: ModelSpec;
  rule?: RuleSpec;
  skill?: SkillSpec;
  mcp?: McpSpec;
};

export type ResourceConfigPatchRequest = {
  name?: string;
  enabled?: boolean;
  model?: ModelSpec;
  rule?: RuleSpec;
  skill?: SkillSpec;
  mcp?: McpSpec;
};

export type ModelTestResult = {
  config_id: string;
  status: "success" | "failed";
  latency_ms: number;
  error_code?: string;
  message: string;
  tested_at: string;
};

export type McpConnectResult = {
  config_id: string;
  status: "connected" | "failed";
  tools: string[];
  error_code?: string;
  message: string;
  connected_at: string;
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

export type WorkspaceProjectConfigItem = {
  project_id: string;
  project_name: string;
  config: {
    project_id: string;
    model_config_ids: string[];
    default_model_config_id: string | null;
    rule_ids: string[];
    skill_ids: string[];
    mcp_ids: string[];
    updated_at: string;
  };
};
