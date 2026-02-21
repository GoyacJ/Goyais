import { normalizeHttpError, normalizeUnknownError } from "@/lib/api-error";
import type { ModelCatalogResponse, ProviderKey } from "@/types/modelCatalog";
import type { EventEnvelope } from "@/types/generated";

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

export interface RunCreateRequest {
  project_id: string;
  session_id: string;
  input: string;
  model_config_id: string;
  workspace_path: string;
  options: {
    use_worktree: boolean;
    run_tests?: string;
  };
}

export async function createRun(payload: RunCreateRequest): Promise<{ run_id: string }> {
  return requestJson("/v1/runs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
}

export function subscribeRunEvents(runId: string, onEvent: (event: EventEnvelope) => void): EventSource {
  const source = new EventSource(`${runtimeBaseUrl()}/v1/runs/${runId}/events`);
  source.onmessage = (message) => {
    const parsed = JSON.parse(message.data) as EventEnvelope;
    onEvent(parsed);
  };
  return source;
}

export async function confirmToolCall(run_id: string, call_id: string, approved: boolean): Promise<void> {
  await requestJson("/v1/tool-confirmations", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ run_id, call_id, approved })
  });
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

export async function listModelCatalog(modelConfigId: string): Promise<ModelCatalogResponse> {
  return requestJson(`/v1/model-configs/${encodeURIComponent(modelConfigId)}/models`, undefined, { retries: 1 });
}

export async function listRuns(sessionId: string): Promise<{ runs: Array<Record<string, string>> }> {
  return requestJson(`/v1/runs?session_id=${encodeURIComponent(sessionId)}`, undefined, { retries: 1 });
}

export interface RuntimeSessionSummary {
  session_id: string;
  project_id: string;
  title: string;
  updated_at: string;
  last_run_id?: string;
  last_status?: string;
  last_input_preview?: string;
}

export async function listSessions(projectId: string): Promise<{ sessions: RuntimeSessionSummary[] }> {
  return requestJson(`/v1/sessions?project_id=${encodeURIComponent(projectId)}`, undefined, { retries: 1 });
}

export async function createSession(payload: { project_id: string; title?: string }): Promise<{ session: RuntimeSessionSummary }> {
  return requestJson("/v1/sessions", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
}

export async function renameSession(sessionId: string, title: string): Promise<{ session: RuntimeSessionSummary }> {
  return requestJson(`/v1/sessions/${encodeURIComponent(sessionId)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title })
  });
}

export async function replayRunEvents(runId: string): Promise<{ events: EventEnvelope[] }> {
  return requestJson(`/v1/runs/${runId}/events/replay`, undefined, { retries: 1 });
}

export async function syncNow(): Promise<Record<string, number>> {
  return requestJson("/v1/sync/now", { method: "POST" }, { retries: 1 });
}

export async function runtimeHealth(): Promise<{ ok: boolean }> {
  return requestJson("/v1/health", undefined, { retries: 1 });
}
