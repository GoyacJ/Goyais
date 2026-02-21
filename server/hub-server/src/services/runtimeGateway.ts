import { type FastifyRequest } from "fastify";

import type { HubDatabase, WorkspaceRuntimeRecord } from "../db";
import { HubServerError } from "../errors";

interface ResolveRuntimeOptions {
  db: HubDatabase;
  workspaceId: string;
  traceId: string;
  runtimeSharedSecret: string;
}

interface RuntimeHealthPayload {
  workspace_id?: unknown;
  [key: string]: unknown;
}

function normalizeRuntimeSharedSecret(secret: string): string {
  const normalized = secret.trim();
  if (!normalized) {
    throw new HubServerError({
      code: "E_INTERNAL",
      message: "Runtime gateway shared secret is not configured.",
      retryable: false,
      statusCode: 500,
      details: {
        config_key: "GOYAIS_HUB_RUNTIME_SHARED_SECRET"
      },
      causeType: "runtime_shared_secret_missing"
    });
  }
  return normalized;
}

async function fetchRuntimeHealth(
  runtimeBaseUrl: string,
  traceId: string,
  runtimeSharedSecret: string
): Promise<{ payload: RuntimeHealthPayload; heartbeatAt: string }> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 2_500);

  try {
    const response = await fetch(`${runtimeBaseUrl}/v1/health`, {
      method: "GET",
      headers: {
        "X-Trace-Id": traceId,
        "X-Hub-Auth": runtimeSharedSecret
      },
      signal: controller.signal
    });
    if (!response.ok) {
      throw new HubServerError({
        code: "E_RUNTIME_OFFLINE",
        message: "Runtime is offline.",
        retryable: true,
        statusCode: 503,
        details: {
          upstream_status: response.status
        },
        causeType: "runtime_health_status"
      });
    }

    const payload = (await response.json()) as RuntimeHealthPayload;
    return {
      payload,
      heartbeatAt: new Date().toISOString()
    };
  } catch (error) {
    if (error instanceof HubServerError) {
      throw error;
    }

    throw new HubServerError({
      code: "E_RUNTIME_OFFLINE",
      message: "Runtime is offline.",
      retryable: true,
      statusCode: 503,
      causeType: "runtime_health_unreachable"
    });
  } finally {
    clearTimeout(timeout);
  }
}

function normalizeRuntimeBaseUrl(runtime: WorkspaceRuntimeRecord): string {
  return runtime.runtime_base_url.trim().replace(/\/+$/, "");
}

export async function resolveRuntimeForWorkspace(options: ResolveRuntimeOptions): Promise<{
  runtime: WorkspaceRuntimeRecord;
  runtimeBaseUrl: string;
  healthPayload: RuntimeHealthPayload;
}> {
  const runtime = options.db.getWorkspaceRuntime(options.workspaceId);
  if (!runtime) {
    throw new HubServerError({
      code: "E_RUNTIME_NOT_CONFIGURED",
      message: "Runtime is not configured for this workspace.",
      retryable: false,
      statusCode: 404,
      causeType: "runtime_registry_missing"
    });
  }

  const runtimeBaseUrl = normalizeRuntimeBaseUrl(runtime);
  const sharedSecret = normalizeRuntimeSharedSecret(options.runtimeSharedSecret);
  let payload: RuntimeHealthPayload;
  let heartbeatAt: string;
  try {
    const probe = await fetchRuntimeHealth(runtimeBaseUrl, options.traceId, sharedSecret);
    payload = probe.payload;
    heartbeatAt = probe.heartbeatAt;
  } catch (error) {
    if (error instanceof HubServerError && error.code === "E_RUNTIME_OFFLINE") {
      options.db.setWorkspaceRuntimeStatus({
        workspaceId: options.workspaceId,
        runtimeStatus: "offline",
        lastHeartbeatAt: null
      });
    }
    throw error;
  }
  const upstreamWorkspaceId = typeof payload.workspace_id === "string" ? payload.workspace_id.trim() : "";

  if (!upstreamWorkspaceId || upstreamWorkspaceId !== options.workspaceId) {
    options.db.setWorkspaceRuntimeStatus({
      workspaceId: options.workspaceId,
      runtimeStatus: "offline",
      lastHeartbeatAt: null
    });

    throw new HubServerError({
      code: "E_RUNTIME_MISCONFIGURED",
      message: "Runtime workspace binding mismatch.",
      retryable: false,
      statusCode: 409,
      details: {
        expected_workspace_id: options.workspaceId,
        upstream_workspace_id: upstreamWorkspaceId || null
      },
      causeType: "runtime_workspace_mismatch"
    });
  }

  options.db.setWorkspaceRuntimeStatus({
    workspaceId: options.workspaceId,
    runtimeStatus: "online",
    lastHeartbeatAt: heartbeatAt
  });

  return {
    runtime,
    runtimeBaseUrl,
    healthPayload: payload
  };
}

interface ForwardRuntimeRequestOptions {
  runtimeBaseUrl: string;
  runtimePath: string;
  method: "GET" | "POST" | "PATCH" | "DELETE";
  userId: string;
  traceId: string;
  runtimeSharedSecret: string;
  payload?: unknown;
  extraHeaders?: Record<string, string>;
  timeoutMs?: number;
}

export async function forwardRuntimeRequest(options: ForwardRuntimeRequestOptions): Promise<Response> {
  const sharedSecret = normalizeRuntimeSharedSecret(options.runtimeSharedSecret);
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 30_000);

  try {
    const headers = new Headers({
      "X-Hub-Auth": sharedSecret,
      "X-User-Id": options.userId,
      "X-Trace-Id": options.traceId
    });
    if (options.extraHeaders) {
      Object.entries(options.extraHeaders).forEach(([key, value]) => {
        headers.set(key, value);
      });
    }

    let body: string | undefined;
    if (options.payload !== undefined) {
      headers.set("Content-Type", "application/json");
      body = JSON.stringify(options.payload);
    }

    return await fetch(`${options.runtimeBaseUrl}${options.runtimePath}`, {
      method: options.method,
      headers,
      body,
      signal: controller.signal
    });
  } catch {
    throw new HubServerError({
      code: "E_RUNTIME_UPSTREAM",
      message: "Runtime upstream request failed.",
      retryable: true,
      statusCode: 502,
      causeType: "runtime_upstream_network"
    });
  } finally {
    clearTimeout(timeout);
  }
}

export async function forwardRuntimeJsonOrThrow(
  response: Response,
  request: FastifyRequest,
  routeHint: string
): Promise<unknown> {
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json")
    ? ((await response.json()) as unknown)
    : {
        ok: response.ok
      };

  if (!response.ok) {
    throw new HubServerError({
      code: "E_RUNTIME_UPSTREAM",
      message: "Runtime upstream request failed.",
      retryable: response.status >= 500,
      statusCode: 502,
      details: {
        trace_id: request.trace_id,
        upstream_status: response.status,
        route: routeHint,
        upstream: payload
      },
      causeType: "runtime_upstream_status"
    });
  }

  return payload;
}
