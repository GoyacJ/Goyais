import { invoke } from "@tauri-apps/api/core";

import {
  createDefaultLocalProcessConfig,
  type LocalProcessConfigV1
} from "@/types/localProcessConfig";

export type LocalServiceName = "hub" | "runtime";

function isTauriRuntime(): boolean {
  if (typeof window === "undefined") {
    return false;
  }
  return Object.prototype.hasOwnProperty.call(window, "__TAURI_INTERNALS__");
}

export async function localConfigRead(): Promise<LocalProcessConfigV1> {
  if (!isTauriRuntime()) {
    return createDefaultLocalProcessConfig();
  }

  try {
    const payload = await invoke<LocalProcessConfigV1>("local_config_read");
    if (!payload || payload.version !== 1) {
      return createDefaultLocalProcessConfig();
    }
    return payload;
  } catch {
    return createDefaultLocalProcessConfig();
  }
}

export async function localConfigWrite(config: LocalProcessConfigV1): Promise<LocalProcessConfigV1> {
  if (!isTauriRuntime()) {
    return config;
  }
  return invoke<LocalProcessConfigV1>("local_config_write", { config });
}

export async function serviceStart(params: {
  service: LocalServiceName;
  command: string;
  cwd: string;
  env?: Record<string, string>;
}): Promise<number> {
  return invoke<number>("service_start", {
    service: params.service,
    command: params.command,
    cwd: params.cwd,
    env: params.env ?? {}
  });
}

export async function serviceStatus(service: LocalServiceName): Promise<number | null> {
  return invoke<number | null>("service_status", { service });
}

export async function serviceStop(service: LocalServiceName): Promise<void> {
  await invoke("service_stop", { service });
}
