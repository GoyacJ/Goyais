import type { EventEnvelope } from "../types/generated";

const RUNTIME_URL = import.meta.env.VITE_RUNTIME_URL ?? "http://127.0.0.1:8040";

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
  const res = await fetch(`${RUNTIME_URL}/v1/runs`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export function subscribeRunEvents(runId: string, onEvent: (event: EventEnvelope) => void): EventSource {
  const source = new EventSource(`${RUNTIME_URL}/v1/runs/${runId}/events`);
  source.onmessage = (message) => {
    const parsed = JSON.parse(message.data) as EventEnvelope;
    onEvent(parsed);
  };
  return source;
}

export async function confirmToolCall(run_id: string, call_id: string, approved: boolean): Promise<void> {
  const res = await fetch(`${RUNTIME_URL}/v1/tool-confirmations`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ run_id, call_id, approved })
  });
  if (!res.ok) throw new Error(await res.text());
}

export async function listProjects(): Promise<{ projects: Array<Record<string, string>> }> {
  const res = await fetch(`${RUNTIME_URL}/v1/projects`);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function createProject(payload: { project_id?: string; name?: string; workspace_path: string }) {
  const res = await fetch(`${RUNTIME_URL}/v1/projects`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function listModelConfigs(): Promise<{ model_configs: Array<Record<string, string>> }> {
  const res = await fetch(`${RUNTIME_URL}/v1/model-configs`);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function createModelConfig(payload: {
  model_config_id?: string;
  provider: string;
  model: string;
  base_url?: string;
  temperature?: number;
  max_tokens?: number;
  secret_ref: string;
}) {
  const res = await fetch(`${RUNTIME_URL}/v1/model-configs`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function listRuns(sessionId: string): Promise<{ runs: Array<Record<string, string>> }> {
  const res = await fetch(`${RUNTIME_URL}/v1/runs?session_id=${encodeURIComponent(sessionId)}`);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function replayRunEvents(runId: string): Promise<{ events: EventEnvelope[] }> {
  const res = await fetch(`${RUNTIME_URL}/v1/runs/${runId}/events/replay`);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function syncNow(): Promise<Record<string, number>> {
  const res = await fetch(`${RUNTIME_URL}/v1/sync/now`, {
    method: "POST"
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}
