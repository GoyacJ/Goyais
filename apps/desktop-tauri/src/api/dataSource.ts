import * as hubClient from "@/api/hubClient";
import { resolveHubContext } from "@/api/sessionDataSource";
import { ApiError } from "@/lib/api-error";
import type { WorkspaceProfile } from "@/stores/workspaceStore";
import type { ModelCatalogResponse, ProviderKey } from "@/types/modelCatalog";
import { isProviderKey } from "@/types/modelCatalog";

function toError(shape: {
  code: string;
  message: string;
  retryable?: boolean;
  status?: number;
}): ApiError {
  return new ApiError({
    code: shape.code,
    message: shape.message,
    retryable: shape.retryable ?? false,
    status: shape.status
  });
}

async function resolveContext(profile: WorkspaceProfile | undefined): Promise<{
  workspaceId: string;
  serverUrl: string;
  token: string;
}> {
  const ctx = await resolveHubContext(profile);
  if (!ctx.workspaceId) {
    throw toError({
      code: "E_VALIDATION",
      message: "Workspace is not selected",
      status: 400
    });
  }
  return {
    workspaceId: ctx.workspaceId,
    serverUrl: ctx.serverUrl,
    token: ctx.token
  };
}

function parseOptionalNumber(value: unknown): number | null {
  if (typeof value === "number") {
    return Number.isFinite(value) ? value : null;
  }

  if (typeof value === "string" && value.trim().length > 0) {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : null;
  }

  return null;
}

function parseProvider(value: unknown): ProviderKey {
  if (typeof value === "string" && isProviderKey(value)) {
    return value;
  }
  return "openai";
}

export interface DataProject {
  project_id: string;
  workspace_id: string | null;
  name: string;
  root_uri: string | null;
  workspace_path: string | null;
  repo_url: string | null;
  branch: string | null;
  sync_status: "pending" | "syncing" | "ready" | "error" | null;
  sync_error: string | null;
  last_synced_at: string | null;
  created_at: string | null;
  updated_at: string | null;
  source: "local" | "remote";
}

export interface ProjectsClient {
  kind: "local" | "remote";
  supportsDelete: boolean;
  supportsGit: boolean;
  list: () => Promise<DataProject[]>;
  create: (input: { name: string; location: string }) => Promise<void>;
  createGit: (input: { name: string; repo_url: string; branch?: string; auth_ref?: string }) => Promise<void>;
  delete: (projectId: string) => Promise<void>;
  sync: (projectId: string) => Promise<void>;
}

export function getProjectsClient(profile: WorkspaceProfile | undefined): ProjectsClient {
  const kind = !profile || profile.kind === "local" ? "local" : "remote";

  return {
    kind,
    supportsDelete: true,
    supportsGit: true,
    list: async () => {
      const ctx = await resolveContext(profile);
      const payload = await hubClient.listProjects(ctx.serverUrl, ctx.token, ctx.workspaceId);
      return payload.projects.map((project) => ({
        project_id: project.project_id,
        workspace_id: project.workspace_id ?? null,
        name: project.name,
        root_uri: project.root_uri ?? null,
        workspace_path: project.root_uri ?? null,
        repo_url: project.repo_url ?? null,
        branch: project.branch ?? null,
        sync_status: project.sync_status ?? null,
        sync_error: project.sync_error ?? null,
        last_synced_at: project.last_synced_at ?? null,
        created_at: project.created_at,
        updated_at: project.updated_at,
        source: kind
      }));
    },
    create: async ({ name, location }) => {
      const ctx = await resolveContext(profile);
      await hubClient.createProject(ctx.serverUrl, ctx.token, ctx.workspaceId, { name, root_uri: location });
    },
    createGit: async ({ name, repo_url, branch, auth_ref }) => {
      const ctx = await resolveContext(profile);
      await hubClient.createProject(ctx.serverUrl, ctx.token, ctx.workspaceId, { name, repo_url, branch, auth_ref });
    },
    delete: async (projectId) => {
      const ctx = await resolveContext(profile);
      await hubClient.deleteProject(ctx.serverUrl, ctx.token, ctx.workspaceId, projectId);
    },
    sync: async (projectId) => {
      const ctx = await resolveContext(profile);
      await hubClient.syncProject(ctx.serverUrl, ctx.token, ctx.workspaceId, projectId);
    }
  };
}

export interface DataModelConfig {
  model_config_id: string;
  workspace_id: string | null;
  provider: ProviderKey;
  model: string;
  base_url: string | null;
  temperature: number | null;
  max_tokens: number | null;
  secret_ref: string;
  created_at: string | null;
  updated_at: string | null;
  source: "local" | "remote";
}

export interface CreateModelConfigInput {
  model_config_id?: string;
  provider: ProviderKey;
  model: string;
  base_url?: string | null;
  temperature?: number | null;
  max_tokens?: number | null;
  secret_ref?: string;
  api_key?: string;
}

export interface UpdateModelConfigInput {
  provider?: ProviderKey;
  model?: string;
  base_url?: string | null;
  temperature?: number | null;
  max_tokens?: number | null;
  secret_ref?: string;
  api_key?: string;
}

export interface ModelConfigsClient {
  kind: "local" | "remote";
  supportsWrite: boolean;
  supportsDelete: boolean;
  supportsModelCatalog: boolean;
  list: () => Promise<DataModelConfig[]>;
  create: (input: CreateModelConfigInput) => Promise<void>;
  update: (modelConfigId: string, input: UpdateModelConfigInput) => Promise<void>;
  delete: (modelConfigId: string) => Promise<void>;
  listModels: (modelConfigId: string, options?: { apiKeyOverride?: string }) => Promise<ModelCatalogResponse>;
}

export function getModelConfigsClient(profile: WorkspaceProfile | undefined): ModelConfigsClient {
  const kind = !profile || profile.kind === "local" ? "local" : "remote";

  return {
    kind,
    supportsWrite: true,
    supportsDelete: true,
    supportsModelCatalog: true,
    list: async () => {
      const ctx = await resolveContext(profile);
      const payload = await hubClient.listModelConfigs(ctx.serverUrl, ctx.token, ctx.workspaceId);
      return payload.model_configs.map((item) => ({
        model_config_id: item.model_config_id,
        workspace_id: item.workspace_id,
        provider: parseProvider(item.provider),
        model: item.model,
        base_url: item.base_url,
        temperature: parseOptionalNumber(item.temperature),
        max_tokens: parseOptionalNumber(item.max_tokens),
        secret_ref: item.secret_ref,
        created_at: item.created_at,
        updated_at: item.updated_at,
        source: kind
      }));
    },
    create: async (input) => {
      if (!input.api_key?.trim()) {
        throw toError({
          code: "E_VALIDATION",
          message: "api_key is required",
          status: 400
        });
      }
      const ctx = await resolveContext(profile);
      await hubClient.createModelConfig(ctx.serverUrl, ctx.token, ctx.workspaceId, {
        provider: input.provider,
        model: input.model,
        base_url: input.base_url ?? null,
        temperature: input.temperature ?? undefined,
        max_tokens: input.max_tokens ?? null,
        api_key: input.api_key.trim()
      });
    },
    update: async (modelConfigId, input) => {
      const ctx = await resolveContext(profile);
      await hubClient.updateModelConfig(ctx.serverUrl, ctx.token, ctx.workspaceId, modelConfigId, {
        provider: input.provider,
        model: input.model,
        base_url: input.base_url ?? null,
        temperature: input.temperature ?? undefined,
        max_tokens: input.max_tokens ?? null,
        api_key: input.api_key?.trim() || undefined
      });
    },
    delete: async (modelConfigId) => {
      const ctx = await resolveContext(profile);
      await hubClient.deleteModelConfig(ctx.serverUrl, ctx.token, ctx.workspaceId, modelConfigId);
    },
    listModels: async (modelConfigId, options) => {
      const ctx = await resolveContext(profile);
      return hubClient.listRuntimeModelCatalog(
        ctx.serverUrl,
        ctx.token,
        ctx.workspaceId,
        modelConfigId,
        options
      );
    }
  };
}
