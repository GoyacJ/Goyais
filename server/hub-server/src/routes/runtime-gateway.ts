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

const sessionParamsSchema = z.object({
  session_id: z.string().min(1)
});

const modelConfigParamsSchema = z.object({
  model_config_id: z.string().min(1)
});

const sessionListQuerySchema = z.object({
  project_id: z.string().min(1)
});

const sessionCreatePayloadSchema = z.object({
  project_id: z.string().min(1),
  title: z.string().min(1).optional()
});

const sessionRenamePayloadSchema = z.object({
  title: z.string().min(1)
});

interface RegisterRuntimeGatewayRoutesOptions {
  db: HubDatabase;
  runtimeSharedSecret: string;
}

export function registerRuntimeGatewayRoutes(
  app: FastifyInstance,
  options: RegisterRuntimeGatewayRoutesOptions
): void {
  app.get("/v1/runtime/model-configs/:model_config_id/models", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "modelconfig:read");

    const paramsParsed = modelConfigParamsSchema.safeParse(request.params);
    if (!paramsParsed.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid model config route params.",
        retryable: false,
        statusCode: 400,
        details: { issues: paramsParsed.error.issues },
        causeType: "runtime_model_config_params"
      });
    }

    const apiKeyOverride = String(request.headers["x-api-key-override"] ?? "").trim();
    const runtimeTarget = await resolveRuntimeForWorkspace({
      db: options.db,
      workspaceId,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret
    });

    const upstream = await forwardRuntimeRequest({
      runtimeBaseUrl: runtimeTarget.runtimeBaseUrl,
      runtimePath: `/v1/model-configs/${encodeURIComponent(paramsParsed.data.model_config_id)}/models`,
      method: "GET",
      userId: user.user_id,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret,
      extraHeaders: apiKeyOverride ? { "X-Api-Key-Override": apiKeyOverride } : undefined
    });

    return await forwardRuntimeJsonOrThrow(upstream, request, "GET /v1/model-configs/:model_config_id/models");
  });

  app.get("/v1/runtime/sessions", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "run:read");

    const parsedQuery = sessionListQuerySchema.safeParse(request.query);
    if (!parsedQuery.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "project_id query parameter is required.",
        retryable: false,
        statusCode: 400,
        details: { issues: parsedQuery.error.issues },
        causeType: "runtime_sessions_query"
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
      runtimePath: `/v1/sessions?project_id=${encodeURIComponent(parsedQuery.data.project_id)}`,
      method: "GET",
      userId: user.user_id,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret
    });

    return await forwardRuntimeJsonOrThrow(upstream, request, "GET /v1/sessions");
  });

  app.post("/v1/runtime/sessions", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "run:create");

    const parsedPayload = sessionCreatePayloadSchema.safeParse(request.body);
    if (!parsedPayload.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid session payload.",
        retryable: false,
        statusCode: 400,
        details: { issues: parsedPayload.error.issues },
        causeType: "runtime_session_payload"
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
      runtimePath: "/v1/sessions",
      method: "POST",
      userId: user.user_id,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret,
      payload: parsedPayload.data
    });

    return await forwardRuntimeJsonOrThrow(upstream, request, "POST /v1/sessions");
  });

  app.patch("/v1/runtime/sessions/:session_id", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "run:create");

    const parsedParams = sessionParamsSchema.safeParse(request.params);
    if (!parsedParams.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid session route params.",
        retryable: false,
        statusCode: 400,
        details: { issues: parsedParams.error.issues },
        causeType: "runtime_session_params"
      });
    }

    const parsedPayload = sessionRenamePayloadSchema.safeParse(request.body);
    if (!parsedPayload.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid session rename payload.",
        retryable: false,
        statusCode: 400,
        details: { issues: parsedPayload.error.issues },
        causeType: "runtime_session_rename_payload"
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
      runtimePath: `/v1/sessions/${encodeURIComponent(parsedParams.data.session_id)}`,
      method: "PATCH",
      userId: user.user_id,
      traceId: request.trace_id,
      runtimeSharedSecret: options.runtimeSharedSecret,
      payload: parsedPayload.data
    });

    return await forwardRuntimeJsonOrThrow(upstream, request, "PATCH /v1/sessions/:session_id");
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
