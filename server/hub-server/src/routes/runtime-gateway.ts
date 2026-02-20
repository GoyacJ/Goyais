import { randomUUID } from "node:crypto";
import { Readable } from "node:stream";

import { z } from "zod";

import type { FastifyInstance } from "fastify";

import {
  requireDomainAuth,
  requirePermission,
  requireWorkspaceIdQuery,
  requireWorkspaceMember
} from "../auth/workspace-rbac";
import type { HubDatabase } from "../db";
import { HubServerError } from "../errors";
import {
  forwardRuntimeJsonOrThrow,
  forwardRuntimeRequest,
  resolveRuntimeForWorkspace
} from "../services/runtimeGateway";

const runEventsParamsSchema = z.object({
  run_id: z.string().min(1)
});

const runListQuerySchema = z.object({
  session_id: z.string().min(1)
});

const toolConfirmationPayloadSchema = z.object({
  run_id: z.string().min(1),
  call_id: z.string().min(1),
  approved: z.boolean()
});

interface RegisterRuntimeGatewayRoutesOptions {
  db: HubDatabase;
  runtimeSharedSecret: string;
}

function buildRunPath(runId: string, suffix: "events" | "replay"): string {
  if (suffix === "events") {
    return `/v1/runs/${encodeURIComponent(runId)}/events`;
  }
  return `/v1/runs/${encodeURIComponent(runId)}/events/replay`;
}

export function registerRuntimeGatewayRoutes(
  app: FastifyInstance,
  options: RegisterRuntimeGatewayRoutesOptions
): void {
  app.post("/v1/runtime/runs", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "run:create");

    const runtimeTarget = await resolveRuntimeForWorkspace({
      db: options.db,
      workspaceId,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret
    });

    const upstream = await forwardRuntimeRequest({
      runtimeBaseUrl: runtimeTarget.runtimeBaseUrl,
      runtimePath: "/v1/runs",
      method: "POST",
      userId: user.user_id,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret,
      payload: request.body
    });

    const payload = (await forwardRuntimeJsonOrThrow(upstream, request, "POST /v1/runs")) as {
      run_id?: unknown;
      status?: unknown;
    };
    if (typeof payload.run_id === "string" && payload.run_id.trim().length > 0) {
      options.db.insertRunIndex({
        runId: payload.run_id,
        workspaceId,
        createdBy: user.user_id,
        status: typeof payload.status === "string" ? payload.status : "running",
        traceId: request.trace_id
      });
    }

    return payload;
  });

  app.get("/v1/runtime/runs", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "run:read");

    const parsedQuery = runListQuerySchema.safeParse(request.query);
    if (!parsedQuery.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "session_id query parameter is required.",
        retryable: false,
        statusCode: 400,
        details: { issues: parsedQuery.error.issues },
        causeType: "runtime_runs_query"
      });
    }

    const runtimeTarget = await resolveRuntimeForWorkspace({
      db: options.db,
      workspaceId,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret
    });

    const upstream = await forwardRuntimeRequest({
      runtimeBaseUrl: runtimeTarget.runtimeBaseUrl,
      runtimePath: `/v1/runs?session_id=${encodeURIComponent(parsedQuery.data.session_id)}`,
      method: "GET",
      userId: user.user_id,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret
    });

    return await forwardRuntimeJsonOrThrow(upstream, request, "GET /v1/runs");
  });

  app.get("/v1/runtime/runs/:run_id/events/replay", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "run:read");

    const paramsParsed = runEventsParamsSchema.safeParse(request.params);
    if (!paramsParsed.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid run route params.",
        retryable: false,
        statusCode: 400,
        details: { issues: paramsParsed.error.issues },
        causeType: "runtime_run_params"
      });
    }

    const runtimeTarget = await resolveRuntimeForWorkspace({
      db: options.db,
      workspaceId,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret
    });

    const upstream = await forwardRuntimeRequest({
      runtimeBaseUrl: runtimeTarget.runtimeBaseUrl,
      runtimePath: buildRunPath(paramsParsed.data.run_id, "replay"),
      method: "GET",
      userId: user.user_id,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret
    });

    return await forwardRuntimeJsonOrThrow(upstream, request, "GET /v1/runs/:run_id/events/replay");
  });

  app.get("/v1/runtime/runs/:run_id/events", async (request, reply) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "run:read");

    const paramsParsed = runEventsParamsSchema.safeParse(request.params);
    if (!paramsParsed.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid run route params.",
        retryable: false,
        statusCode: 400,
        details: { issues: paramsParsed.error.issues },
        causeType: "runtime_run_params"
      });
    }

    const runtimeTarget = await resolveRuntimeForWorkspace({
      db: options.db,
      workspaceId,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret
    });

    const upstream = await forwardRuntimeRequest({
      runtimeBaseUrl: runtimeTarget.runtimeBaseUrl,
      runtimePath: buildRunPath(paramsParsed.data.run_id, "events"),
      method: "GET",
      userId: user.user_id,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret,
      timeoutMs: 5 * 60 * 1000
    });

    if (!upstream.ok) {
      await forwardRuntimeJsonOrThrow(upstream, request, "GET /v1/runs/:run_id/events");
      return;
    }

    if (!upstream.body) {
      throw new HubServerError({
        code: "E_RUNTIME_UPSTREAM",
        message: "Runtime SSE stream is unavailable.",
        retryable: true,
        statusCode: 502,
        causeType: "runtime_sse_body_missing"
      });
    }

    const contentType = upstream.headers.get("content-type") ?? "text/event-stream";
    reply.header("Content-Type", contentType);
    reply.header("Cache-Control", "no-cache");
    reply.header("Connection", "keep-alive");
    return reply.send(Readable.fromWeb(upstream.body as unknown as any));
  });

  app.post("/v1/runtime/tool-confirmations", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "confirm:write");

    const parsedPayload = toolConfirmationPayloadSchema.safeParse(request.body);
    if (!parsedPayload.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid tool confirmation payload.",
        retryable: false,
        statusCode: 400,
        details: { issues: parsedPayload.error.issues },
        causeType: "runtime_confirmation_payload"
      });
    }

    const runtimeTarget = await resolveRuntimeForWorkspace({
      db: options.db,
      workspaceId,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret
    });

    const upstream = await forwardRuntimeRequest({
      runtimeBaseUrl: runtimeTarget.runtimeBaseUrl,
      runtimePath: "/v1/tool-confirmations",
      method: "POST",
      userId: user.user_id,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret,
      payload: parsedPayload.data
    });

    const payload = await forwardRuntimeJsonOrThrow(upstream, request, "POST /v1/tool-confirmations");
    options.db.insertAuditIndex({
      auditId: randomUUID(),
      workspaceId,
      runId: parsedPayload.data.run_id,
      userId: user.user_id,
      action: "tool_confirmation",
      toolName: null,
      outcome: parsedPayload.data.approved ? "approved" : "denied"
    });

    return payload;
  });

  app.get("/v1/runtime/health", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "workspace:read");

    const runtimeTarget = await resolveRuntimeForWorkspace({
      db: options.db,
      workspaceId,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret
    });

    return {
      workspace_id: workspaceId,
      runtime_base_url: runtimeTarget.runtimeBaseUrl,
      runtime_status: "online",
      upstream: runtimeTarget.healthPayload
    };
  });

  app.get("/v1/runtime/version", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "workspace:read");

    const runtimeTarget = await resolveRuntimeForWorkspace({
      db: options.db,
      workspaceId,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret
    });

    const upstream = await forwardRuntimeRequest({
      runtimeBaseUrl: runtimeTarget.runtimeBaseUrl,
      runtimePath: "/v1/version",
      method: "GET",
      userId: user.user_id,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret
    });

    return await forwardRuntimeJsonOrThrow(upstream, request, "GET /v1/version");
  });
}
