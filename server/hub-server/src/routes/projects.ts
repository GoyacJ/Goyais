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

const createProjectSchema = z.object({
  name: z.string().min(1),
  root_uri: z.string().min(1)
});

const projectParamsSchema = z.object({
  project_id: z.string().min(1)
});

interface RegisterProjectRoutesOptions {
  db: HubDatabase;
}

export function registerProjectRoutes(app: FastifyInstance, options: RegisterProjectRoutesOptions): void {
  app.get("/v1/projects", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "project:read");

    const projects = options.db.listProjects(workspaceId);
    return { projects };
  });

  app.post("/v1/projects", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "project:write");

    const parsed = createProjectSchema.safeParse(request.body);
    if (!parsed.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid project payload.",
        retryable: false,
        statusCode: 400,
        details: { issues: parsed.error.issues },
        causeType: "project_payload"
      });
    }

    const project = options.db.createProject({
      workspaceId,
      name: parsed.data.name.trim(),
      rootUri: parsed.data.root_uri.trim(),
      createdBy: user.user_id
    });

    return { project };
  });

  app.delete("/v1/projects/:project_id", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "project:write");

    const paramsParsed = projectParamsSchema.safeParse(request.params);
    if (!paramsParsed.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid project route params.",
        retryable: false,
        statusCode: 400,
        details: { issues: paramsParsed.error.issues },
        causeType: "project_params"
      });
    }

    const deleted = options.db.deleteProject(workspaceId, paramsParsed.data.project_id);
    if (!deleted) {
      throw new HubServerError({
        code: "E_NOT_FOUND",
        message: "Project not found.",
        retryable: false,
        statusCode: 404,
        causeType: "project_lookup"
      });
    }

    return {
      ok: true
    };
  });
}
