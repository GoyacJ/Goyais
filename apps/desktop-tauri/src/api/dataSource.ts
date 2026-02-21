import * as hubClient from "@/api/hubClient";
import * as runtimeClient from "@/api/runtimeClient";
import { loadToken } from "@/api/secretStoreClient";
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

async function resolveRemoteContext(profile: WorkspaceProfile): Promise<{
  workspaceId: string;
  serverUrl: string;
  token: string;
}> {
  if (profile.kind !== "remote" || !profile.remote) {
    throw toError({
      code: "E_VALIDATION",
      message: "Remote workspace profile is required",
      status: 400
    });
  }

  const workspaceId = profile.remote.selectedWorkspaceId;
  if (!workspaceId) {
    throw toError({
      code: "E_VALIDATION",
      message: "Remote workspace is not selected",
      status: 400
    });
  }

  const tokenRef = profile.remote.tokenRef || profile.id;
  const token = await loadToken(tokenRef);
  if (!token) {
    throw toError({
      code: "E_UNAUTHORIZED",
      message: "Token not found in keychain. Please login again.",
      status: 401
    });
  }

  return {
    workspaceId,
    serverUrl: profile.remote.serverUrl,
    token
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

const LOCAL_TEST_PROJECT_ID_PATTERNS = [/^project-sessions/i, /^project-rename/i, /^diag-project-diag-execution-/i];

function shouldIgnoreLocalProject(project: Record<string, unknown>): boolean {
  const projectId = String(project.project_id ?? "");
  return LOCAL_TEST_PROJECT_ID_PATTERNS.some((pattern) => pattern.test(projectId));
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
  if (!profile || profile.kind === "local") {
    const notSupported = async () => {
      throw toError({ code: "E_VALIDATION", message: "Local data source does not support this operation", status: 400 });
    };
    return {
      kind: "local",
      supportsDelete: true,
      supportsGit: false,
      list: async () => {
        const payload = await runtimeClient.listProjects();
        const deduped = new Map<string, DataProject>();
        payload.projects
          .filter((project) => !shouldIgnoreLocalProject(project))
          .forEach((project, index) => {
            const projectId = String(project.project_id ?? `local-project-${index}`);
            if (deduped.has(projectId)) {
              return;
            }
            deduped.set(projectId, {
              project_id: projectId,
              workspace_id: null,
              name: String(project.name ?? ""),
              root_uri: null,
              workspace_path: String(project.workspace_path ?? ""),
              repo_url: null,
              branch: null,
              sync_status: null,
              sync_error: null,
              last_synced_at: null,
              created_at: null,
              updated_at: null,
              source: "local"
            });
          });
        return [...deduped.values()];
      },
      create: async ({ name, location }) => {
        await runtimeClient.createProject({ name, workspace_path: location });
      },
      createGit: notSupported,
      delete: async (projectId) => {
        await runtimeClient.deleteProject(projectId);
      },
      sync: notSupported
    };
  }

  return {
    kind: "remote",
    supportsDelete: true,
    supportsGit: true,
    list: async () => {
      const remote = await resolveRemoteContext(profile);
      const payload = await hubClient.listProjects(remote.serverUrl, remote.token, remote.workspaceId);
      return payload.projects.map((project) => ({
        project_id: project.project_id,
        workspace_id: project.workspace_id,
        name: project.name,
        root_uri: project.root_uri ?? null,
        workspace_path: null,
        repo_url: project.repo_url ?? null,
        branch: project.branch ?? null,
        sync_status: project.sync_status ?? null,
        sync_error: project.sync_error ?? null,
        last_synced_at: project.last_synced_at ?? null,
        created_at: project.created_at,
        updated_at: project.updated_at,
        source: "remote"
      }));
    },
    create: async ({ name, location }) => {
      const remote = await resolveRemoteContext(profile);
      await hubClient.createProject(remote.serverUrl, remote.token, remote.workspaceId, { name, root_uri: location });
    },
    createGit: async ({ name, repo_url, branch, auth_ref }) => {
      const remote = await resolveRemoteContext(profile);
      await hubClient.createProject(remote.serverUrl, remote.token, remote.workspaceId, { name, repo_url, branch, auth_ref });
    },
    delete: async (projectId) => {
      const remote = await resolveRemoteContext(profile);
      await hubClient.deleteProject(remote.serverUrl, remote.token, remote.workspaceId, projectId);
    },
    sync: async (projectId) => {
      const remote = await resolveRemoteContext(profile);
      await hubClient.syncProject(remote.serverUrl, remote.token, remote.workspaceId, projectId);
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
  if (!profile || profile.kind === "local") {
    return {
      kind: "local",
      supportsWrite: true,
      supportsDelete: true,
      supportsModelCatalog: true,
      list: async () => {
        const payload = await runtimeClient.listModelConfigs();
        return payload.model_configs.map((item, index) => ({
          model_config_id: String(item.model_config_id ?? `local-model-config-${index}`),
          workspace_id: null,
          provider: parseProvider(item.provider),
          model: String(item.model ?? ""),
          base_url: typeof item.base_url === "string" ? item.base_url : null,
          temperature: parseOptionalNumber(item.temperature),
          max_tokens: parseOptionalNumber(item.max_tokens),
          secret_ref: String(item.secret_ref ?? ""),
          created_at: typeof item.created_at === "string" ? item.created_at : null,
          updated_at: typeof item.updated_at === "string" ? item.updated_at : null,
          source: "local"
        }));
      },
      create: async (input) => {
        const modelConfigId = input.model_config_id ?? crypto.randomUUID();
        await runtimeClient.createModelConfig({
          model_config_id: modelConfigId,
          provider: input.provider,
          model: input.model,
          base_url: input.base_url ?? undefined,
          temperature: input.temperature ?? undefined,
          max_tokens: input.max_tokens ?? undefined,
          secret_ref: input.secret_ref ?? `keychain:${input.provider}:${modelConfigId}`
        });
      },
      update: async (modelConfigId, input) => {
        const secretRef =
          input.secret_ref !== undefined
            ? input.secret_ref
            : input.provider
              ? `keychain:${input.provider}:${modelConfigId}`
              : undefined;
        await runtimeClient.updateModelConfig(modelConfigId, {
          provider: input.provider,
          model: input.model,
          base_url: input.base_url ?? undefined,
          temperature: input.temperature ?? undefined,
          max_tokens: input.max_tokens ?? undefined,
          secret_ref: secretRef
        });
      },
      delete: async (modelConfigId) => {
        await runtimeClient.deleteModelConfig(modelConfigId);
      },
      listModels: async (modelConfigId, options) =>
        runtimeClient.listModelCatalog(modelConfigId, {
          apiKeyOverride: options?.apiKeyOverride
        })
    };
  }

  return {
    kind: "remote",
    supportsWrite: true,
    supportsDelete: true,
    supportsModelCatalog: true,
    list: async () => {
      const remote = await resolveRemoteContext(profile);
      const payload = await hubClient.listModelConfigs(remote.serverUrl, remote.token, remote.workspaceId);
      return payload.model_configs.map((item) => ({
        model_config_id: item.model_config_id,
        workspace_id: item.workspace_id,
        provider: parseProvider(item.provider),
        model: item.model,
        base_url: item.base_url,
        temperature: item.temperature,
        max_tokens: item.max_tokens,
        secret_ref: item.secret_ref,
        created_at: item.created_at,
        updated_at: item.updated_at,
        source: "remote"
      }));
    },
    create: async (input) => {
      if (!input.api_key) {
        throw toError({
          code: "E_VALIDATION",
          message: "api_key is required for remote model configs",
          status: 400
        });
      }

      const remote = await resolveRemoteContext(profile);
      await hubClient.createModelConfig(remote.serverUrl, remote.token, remote.workspaceId, {
        provider: input.provider,
        model: input.model,
        base_url: input.base_url ?? null,
        temperature: input.temperature ?? undefined,
        max_tokens: input.max_tokens ?? null,
        api_key: input.api_key
      });
    },
    update: async (modelConfigId, input) => {
      const remote = await resolveRemoteContext(profile);
      await hubClient.updateModelConfig(remote.serverUrl, remote.token, remote.workspaceId, modelConfigId, {
        provider: input.provider,
        model: input.model,
        base_url: input.base_url ?? null,
        temperature: input.temperature ?? undefined,
        max_tokens: input.max_tokens ?? null,
        api_key: input.api_key
      });
    },
    delete: async (modelConfigId) => {
      const remote = await resolveRemoteContext(profile);
      await hubClient.deleteModelConfig(remote.serverUrl, remote.token, remote.workspaceId, modelConfigId);
    },
    listModels: async (modelConfigId) => {
      const remote = await resolveRemoteContext(profile);
      return hubClient.listRuntimeModelCatalog(
        remote.serverUrl,
        remote.token,
        remote.workspaceId,
        modelConfigId
      );
    }
  };
}
