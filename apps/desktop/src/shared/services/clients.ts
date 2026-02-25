import { createApiClient, type ApiClient } from "@/shared/services/http";
import { getControlHubBaseUrl, validateWorkspaceHubUrl } from "@/shared/runtime";
let controlClient: ApiClient | null = null;
let controlClientBaseURL = "";
const targetClients = new Map<string, ApiClient>();

export function getControlClient(): ApiClient {
  const baseURL = getControlHubBaseUrl();
  if (controlClient === null || controlClientBaseURL !== baseURL) {
    controlClient = createApiClient(baseURL);
    controlClientBaseURL = baseURL;
  }
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
  return validateWorkspaceHubUrl(raw);
}
