import { normalizeHttpError, normalizeUnknownError } from "@/lib/api-error";
import type { ModelCatalogResponse, ProviderKey } from "@/types/modelCatalog";

const DEFAULT_RUNTIME_URL = import.meta.env.VITE_RUNTIME_URL ?? "http://127.0.0.1:8040";
const RUNTIME_URL_STORAGE_KEY = "goyais.runtimeUrl";

function runtimeBaseUrl(): string {
  return localStorage.getItem(RUNTIME_URL_STORAGE_KEY) ?? DEFAULT_RUNTIME_URL;
}

interface RequestOptions {
  retries?: number;
  retryDelayMs?: number;
}

async function requestJson<T>(path: string, init?: RequestInit, options?: RequestOptions): Promise<T> {
  const retries = options?.retries ?? 0;
  const retryDelayMs = options?.retryDelayMs ?? 250;

  for (let attempt = 0; attempt <= retries; attempt += 1) {
    try {
      const res = await fetch(`${runtimeBaseUrl()}${path}`, init);
      if (!res.ok) {
        throw await normalizeHttpError(res);
      }

      if (res.status === 204) {
        return undefined as T;
      }

      return (await res.json()) as T;
    } catch (error) {
      const normalized = normalizeUnknownError(error);
      if (attempt < retries && normalized.retryable) {
        await new Promise((resolve) => setTimeout(resolve, retryDelayMs * (attempt + 1)));
        continue;
      }
      throw normalized;
    }
  }

  throw new Error("Unreachable request state");
}

export async function listProjects(): Promise<{ projects: Array<Record<string, string>> }> {
  return requestJson("/v1/projects", undefined, { retries: 1 });
}

export async function createProject(payload: { project_id?: string; name?: string; workspace_path: string }) {
  return requestJson("/v1/projects", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
}

export async function deleteProject(projectId: string) {
  return requestJson(`/v1/projects/${encodeURIComponent(projectId)}`, {
    method: "DELETE"
  });
}

export async function listModelConfigs(): Promise<{ model_configs: Array<Record<string, string>> }> {
  return requestJson("/v1/model-configs", undefined, { retries: 1 });
}

export async function createModelConfig(payload: {
  model_config_id?: string;
  provider: ProviderKey;
  model: string;
  base_url?: string;
  temperature?: number;
  max_tokens?: number;
  secret_ref: string;
}) {
  return requestJson("/v1/model-configs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
}

export async function updateModelConfig(
  modelConfigId: string,
  payload: {
    provider?: ProviderKey;
    model?: string;
    base_url?: string;
    temperature?: number;
    max_tokens?: number;
    secret_ref?: string;
  }
) {
  return requestJson(`/v1/model-configs/${encodeURIComponent(modelConfigId)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
}

export async function deleteModelConfig(modelConfigId: string) {
  return requestJson(`/v1/model-configs/${encodeURIComponent(modelConfigId)}`, {
    method: "DELETE"
  });
}

export async function listModelCatalog(
  modelConfigId: string,
  options?: { apiKeyOverride?: string }
): Promise<ModelCatalogResponse> {
  const init =
    options?.apiKeyOverride
      ? {
          headers: {
            "X-Api-Key-Override": options.apiKeyOverride
          }
        }
      : undefined;
  return requestJson(`/v1/model-configs/${encodeURIComponent(modelConfigId)}/models`, init, { retries: 1 });
}

export async function syncNow(): Promise<Record<string, number>> {
  return requestJson("/v1/sync/now", { method: "POST" }, { retries: 1 });
}

export async function runtimeHealth(): Promise<{ ok: boolean }> {
  return requestJson("/v1/health", undefined, { retries: 1 });
}
