export type LocalAuthMode = "local_open" | "remote_auth";

export interface LocalServiceHubConfig {
  port: string;
  authMode: LocalAuthMode;
  dbDriver: "sqlite" | "postgres";
  dbPath: string;
  databaseUrl: string;
  workerBaseUrl: string;
  maxConcurrentExecutions: string;
  logLevel: string;
  advancedEnv: Record<string, string>;
}

export interface LocalServiceRuntimeConfig {
  host: string;
  port: string;
  agentMode: string;
  hubBaseUrl: string;
  requireHubAuth: boolean;
  workspaceId: string;
  workspaceRoot: string;
  syncServerUrl: string;
  syncDeviceId: string;
  advancedEnv: Record<string, string>;
}

export interface LocalConnectionConfig {
  localHubUrl: string;
  defaultRemoteServerUrl: string;
}

export interface LocalProcessConfigV1 {
  version: 1;
  hub: LocalServiceHubConfig;
  runtime: LocalServiceRuntimeConfig;
  connections: LocalConnectionConfig;
  pendingApply: {
    hub: boolean;
    runtime: boolean;
  };
}

function envValue(key: string): string | undefined {
  const record = (import.meta as ImportMeta & { env?: Record<string, string> }).env;
  return record?.[key];
}

export function normalizeHubUrl(value: string): string {
  const trimmed = value.trim().replace(/\/+$/, "");
  if (!trimmed) {
    return "http://127.0.0.1:8787";
  }
  if (trimmed.startsWith("http://") || trimmed.startsWith("https://")) {
    return trimmed;
  }
  return `http://${trimmed}`;
}

export function createDefaultLocalProcessConfig(): LocalProcessConfigV1 {
  const localHubUrl = normalizeHubUrl(envValue("VITE_LOCAL_HUB_URL") ?? "http://127.0.0.1:8787");
  const runtimePort = "8040";
  const runtimeHost = "127.0.0.1";
  const runtimeBaseUrl = `${runtimeHost}:${runtimePort}`;

  return {
    version: 1,
    hub: {
      port: "8787",
      authMode: "local_open",
      dbDriver: "sqlite",
      dbPath: "./data/hub.db",
      databaseUrl: "",
      workerBaseUrl: `http://${runtimeBaseUrl}`,
      maxConcurrentExecutions: "5",
      logLevel: "info",
      advancedEnv: {}
    },
    runtime: {
      host: runtimeHost,
      port: runtimePort,
      agentMode: "vanilla",
      hubBaseUrl: localHubUrl,
      requireHubAuth: true,
      workspaceId: "local",
      workspaceRoot: ".",
      syncServerUrl: "http://127.0.0.1:8140",
      syncDeviceId: "local-device",
      advancedEnv: {}
    },
    connections: {
      localHubUrl,
      defaultRemoteServerUrl: localHubUrl
    },
    pendingApply: {
      hub: false,
      runtime: false
    }
  };
}

