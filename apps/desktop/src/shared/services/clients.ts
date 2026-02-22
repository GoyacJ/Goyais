import { createApiClient, type ApiClient } from "@/shared/services/http";

const CONTROL_BASE_URL = import.meta.env.VITE_HUB_BASE_URL ?? "http://127.0.0.1:8787";

const controlClient = createApiClient(CONTROL_BASE_URL);
const targetClients = new Map<string, ApiClient>();

export function getControlClient(): ApiClient {
  return controlClient;
}

export function getTargetClient(hubUrl: string): ApiClient {
  const normalizedHubUrl = normalizeBaseURL(hubUrl);

  const cached = targetClients.get(normalizedHubUrl);
  if (cached) {
    return cached;
  }

  const client = createApiClient(normalizedHubUrl);
  targetClients.set(normalizedHubUrl, client);
  return client;
}

function normalizeBaseURL(raw: string): string {
  return raw.trim().replace(/\/$/, "");
}
