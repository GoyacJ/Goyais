import { normalizeHttpError, normalizeUnknownError } from "@/lib/api-error";
import type { ModelCatalogResponse } from "@/types/modelCatalog";
import type { EventEnvelope } from "@/types/generated";

function normalizeServerUrl(serverUrl: string): string {
  return serverUrl.trim().replace(/\/+$/, "");
}

function runtimeBase(serverUrl: string): string {
  return `${normalizeServerUrl(serverUrl)}/v1/runtime`;
}

function authHeaders(token: string): Headers {
  const headers = new Headers();
  headers.set("Authorization", `Bearer ${token}`);
  headers.set("X-Trace-Id", crypto.randomUUID());
  return headers;
}

function withWorkspaceQuery(path: string, workspaceId: string): string {
  const joiner = path.includes("?") ? "&" : "?";
  return `${path}${joiner}workspace_id=${encodeURIComponent(workspaceId)}`;
}

async function requestJson<T>(
  serverUrl: string,
  token: string,
  workspaceId: string,
  path: string,
  init?: RequestInit
): Promise<T> {
  try {
    const headers = authHeaders(token);
    const initHeaders = new Headers(init?.headers ?? {});
    initHeaders.forEach((value, key) => headers.set(key, value));

    const response = await fetch(`${runtimeBase(serverUrl)}${withWorkspaceQuery(path, workspaceId)}`, {
      ...init,
      headers
    });

    if (!response.ok) {
      throw await normalizeHttpError(response);
    }

    return (await response.json()) as T;
  } catch (error) {
    throw normalizeUnknownError(error);
  }
}

export interface RuntimeGatewayRunCreateRequest {
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

export interface RuntimeGatewaySessionSummary {
  session_id: string;
  project_id: string;
  title: string;
  updated_at: string;
  last_run_id?: string;
  last_status?: string;
  last_input_preview?: string;
}

export function createRun(
  serverUrl: string,
  token: string,
  workspaceId: string,
  payload: RuntimeGatewayRunCreateRequest
): Promise<{ run_id: string }> {
  return requestJson(serverUrl, token, workspaceId, "/runs", {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(payload)
  });
}

export async function confirmToolCall(
  serverUrl: string,
  token: string,
  workspaceId: string,
  runId: string,
  callId: string,
  approved: boolean
): Promise<void> {
  await requestJson(serverUrl, token, workspaceId, "/tool-confirmations", {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({
      run_id: runId,
      call_id: callId,
      approved
    })
  });
}

export function listRuns(
  serverUrl: string,
  token: string,
  workspaceId: string,
  sessionId: string
): Promise<{ runs: Array<Record<string, string>> }> {
  return requestJson(serverUrl, token, workspaceId, `/runs?session_id=${encodeURIComponent(sessionId)}`);
}

export function replayRunEvents(
  serverUrl: string,
  token: string,
  workspaceId: string,
  runId: string
): Promise<{ events: EventEnvelope[] }> {
  return requestJson(serverUrl, token, workspaceId, `/runs/${encodeURIComponent(runId)}/events/replay`);
}

export function runtimeHealth(
  serverUrl: string,
  token: string,
  workspaceId: string
): Promise<{ runtime_status: string; upstream: { ok: boolean } }> {
  return requestJson(serverUrl, token, workspaceId, "/health");
}

export function listSessions(
  serverUrl: string,
  token: string,
  workspaceId: string,
  projectId: string
): Promise<{ sessions: RuntimeGatewaySessionSummary[] }> {
  return requestJson(serverUrl, token, workspaceId, `/sessions?project_id=${encodeURIComponent(projectId)}`);
}

export function createSession(
  serverUrl: string,
  token: string,
  workspaceId: string,
  payload: { project_id: string; title?: string }
): Promise<{ session: RuntimeGatewaySessionSummary }> {
  return requestJson(serverUrl, token, workspaceId, "/sessions", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
}

export function renameSession(
  serverUrl: string,
  token: string,
  workspaceId: string,
  sessionId: string,
  title: string
): Promise<{ session: RuntimeGatewaySessionSummary }> {
  return requestJson(serverUrl, token, workspaceId, `/sessions/${encodeURIComponent(sessionId)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title })
  });
}

export function listModelCatalog(
  serverUrl: string,
  token: string,
  workspaceId: string,
  modelConfigId: string
): Promise<ModelCatalogResponse> {
  return requestJson(
    serverUrl,
    token,
    workspaceId,
    `/model-configs/${encodeURIComponent(modelConfigId)}/models`
  );
}

function parseSseData(buffer: string, onEvent: (event: EventEnvelope) => void): string {
  let remaining = buffer;
  while (true) {
    const boundary = remaining.indexOf("\n\n");
    if (boundary < 0) {
      break;
    }

    const rawFrame = remaining.slice(0, boundary);
    remaining = remaining.slice(boundary + 2);

    const dataLines = rawFrame
      .split(/\r?\n/g)
      .filter((line) => line.startsWith("data:"))
      .map((line) => line.slice(5).trimStart());

    if (dataLines.length === 0) {
      continue;
    }

    const jsonPayload = dataLines.join("\n");
    const parsed = JSON.parse(jsonPayload) as EventEnvelope;
    onEvent(parsed);
  }
  return remaining;
}

export interface RuntimeEventSubscription {
  close: () => void;
}

export function subscribeRunEvents(
  serverUrl: string,
  token: string,
  workspaceId: string,
  runId: string,
  onEvent: (event: EventEnvelope) => void,
  onError?: (error: Error) => void
): RuntimeEventSubscription {
  const controller = new AbortController();

  void (async () => {
    try {
      const response = await fetch(
        `${runtimeBase(serverUrl)}${withWorkspaceQuery(`/runs/${encodeURIComponent(runId)}/events`, workspaceId)}`,
        {
          method: "GET",
          headers: authHeaders(token),
          signal: controller.signal
        }
      );

      if (!response.ok) {
        throw await normalizeHttpError(response);
      }
      if (!response.body) {
        throw new Error("Missing event stream body");
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const chunk = await reader.read();
        if (chunk.done) {
          break;
        }
        buffer += decoder.decode(chunk.value, { stream: true });
        buffer = parseSseData(buffer, onEvent);
      }
    } catch (error) {
      if (controller.signal.aborted) {
        return;
      }
      onError?.(normalizeUnknownError(error));
    }
  })();

  return {
    close() {
      controller.abort();
    }
  };
}
