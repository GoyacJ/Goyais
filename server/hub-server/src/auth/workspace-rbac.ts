import type { FastifyRequest } from "fastify";

import { requireAuth, type AuthUser } from "./bearer";
import type { HubDatabase, MembershipRole } from "../db";
import { HubServerError } from "../errors";

declare module "fastify" {
  interface FastifyRequest {
    auth_workspace_id?: string;
    auth_membership?: MembershipRole;
  }
}

function normalizeWorkspaceId(raw: unknown): string | null {
  if (typeof raw === "string" && raw.trim().length > 0) {
    return raw.trim();
  }

  if (Array.isArray(raw) && raw.length > 0 && typeof raw[0] === "string" && raw[0].trim().length > 0) {
    return raw[0].trim();
  }

  return null;
}

export function requireWorkspaceIdQuery(request: FastifyRequest): string {
  const query = (request.query ?? {}) as Record<string, unknown>;
  const workspaceId = normalizeWorkspaceId(query.workspace_id);

  if (!workspaceId) {
    throw new HubServerError({
      code: "E_VALIDATION",
      message: "workspace_id query parameter is required.",
      retryable: false,
      statusCode: 400,
      causeType: "workspace_id_query"
    });
  }

  request.auth_workspace_id = workspaceId;
  return workspaceId;
}

export function requireDomainAuth(request: FastifyRequest, db: HubDatabase): AuthUser {
  try {
    return requireAuth(request, db);
  } catch (error) {
    if (error instanceof HubServerError && error.statusCode === 401) {
      throw new HubServerError({
        code: "E_UNAUTHORIZED",
        message: "Unauthorized.",
        retryable: false,
        statusCode: 401,
        causeType: "domain_auth"
      });
    }

    throw error;
  }
}

export function requireWorkspaceMember(
  request: FastifyRequest,
  db: HubDatabase,
  user: AuthUser,
  workspaceId: string
): MembershipRole {
  const membership = db.getMembershipRole(user.user_id, workspaceId);
  if (!membership) {
    throw new HubServerError({
      code: "E_FORBIDDEN",
      message: "You are not a member of this workspace.",
      retryable: false,
      statusCode: 403,
      causeType: "workspace_membership"
    });
  }

  request.auth_membership = membership;
  return membership;
}

export function requirePermission(db: HubDatabase, roleId: string, permKey: string): void {
  const granted = db.roleHasPermission(roleId, permKey);
  if (!granted) {
    throw new HubServerError({
      code: "E_FORBIDDEN",
      message: "Missing required permission.",
      retryable: false,
      statusCode: 403,
      details: {
        perm_key: permKey
      },
      causeType: "workspace_permission"
    });
  }
}

export function requireWorkspaceManagePermission(db: HubDatabase, roleId: string): void {
  requirePermission(db, roleId, "workspace:manage");
}
