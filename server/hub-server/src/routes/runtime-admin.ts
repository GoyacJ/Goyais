import { z } from "zod";

import type { FastifyInstance } from "fastify";

import {
  requireDomainAuth,
  requireWorkspaceMember,
  requireWorkspaceManagePermission
} from "../auth/workspace-rbac";
import type { HubDatabase } from "../db";
import { HubServerError } from "../errors";

const runtimeRegistryParamsSchema = z.object({
  workspace_id: z.string().min(1)
});

const runtimeRegistryBodySchema = z.object({
  runtime_base_url: z.string().url()
});

interface RegisterRuntimeAdminRoutesOptions {
  db: HubDatabase;
  runtimeSharedSecret?: string;
}

async function detectRuntimeStatus(
  runtimeBaseUrl: string,
  workspaceId: string,
  traceId: string,
  runtimeSharedSecret: string
): Promise<{
  status: "online" | "offline";
  heartbeatAt: string | null;
}> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 2_500);

  try {
    const headers: Record<string, string> = {
      "X-Trace-Id": traceId
    };
    if (runtimeSharedSecret.trim()) {
      headers["X-Hub-Auth"] = runtimeSharedSecret.trim();
    }

    const response = await fetch(`${runtimeBaseUrl}/v1/health`, {
      method: "GET",
      headers,
      signal: controller.signal
    });

    if (!response.ok) {
      return {
        status: "offline",
        heartbeatAt: null
      };
    }

    const payload = (await response.json()) as { workspace_id?: unknown };
    const upstreamWorkspaceId = typeof payload.workspace_id === "string" ? payload.workspace_id.trim() : "";
    if (!upstreamWorkspaceId || upstreamWorkspaceId !== workspaceId) {
      return {
        status: "offline",
        heartbeatAt: null
      };
    }

    return {
      status: "online",
      heartbeatAt: new Date().toISOString()
    };
  } catch {
    return {
      status: "offline",
      heartbeatAt: null
    };
  } finally {
    clearTimeout(timeout);
  }
}

export function registerRuntimeAdminRoutes(app: FastifyInstance, options: RegisterRuntimeAdminRoutesOptions): void {
  app.post("/v1/admin/workspaces/:workspace_id/runtime", async (request) => {
    const user = requireDomainAuth(request, options.db);

    const paramsParsed = runtimeRegistryParamsSchema.safeParse(request.params);
    if (!paramsParsed.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid workspace route params.",
        retryable: false,
        statusCode: 400,
        details: { issues: paramsParsed.error.issues },
        causeType: "runtime_registry_params"
      });
    }

    const bodyParsed = runtimeRegistryBodySchema.safeParse(request.body);
    if (!bodyParsed.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid runtime registry payload.",
        retryable: false,
        statusCode: 400,
        details: { issues: bodyParsed.error.issues },
        causeType: "runtime_registry_payload"
      });
    }

    const workspaceId = paramsParsed.data.workspace_id;
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requireWorkspaceManagePermission(options.db, membership.role_id);

    const runtimeBaseUrl = bodyParsed.data.runtime_base_url.trim().replace(/\/+$/, "");
    const runtimeStatus = await detectRuntimeStatus(
      runtimeBaseUrl,
      workspaceId,
      request.trace_id,
      options.runtimeSharedSecret ?? ""
    );
    const runtime = options.db.upsertWorkspaceRuntime({
      workspaceId,
      runtimeBaseUrl,
      runtimeStatus: runtimeStatus.status,
      lastHeartbeatAt: runtimeStatus.heartbeatAt
    });

    return runtime;
  });
}
