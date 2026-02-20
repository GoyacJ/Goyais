import { z } from "zod";

import type { FastifyInstance } from "fastify";

import { requireAuth } from "../auth/bearer";
import type { HubDatabase } from "../db";
import { HubServerError } from "../errors";
import { buildMenuTree } from "../services/navigation";

const navigationQuerySchema = z.object({
  workspace_id: z.string().min(1)
});

interface RegisterNavigationRoutesOptions {
  db: HubDatabase;
}

export function registerNavigationRoutes(app: FastifyInstance, options: RegisterNavigationRoutesOptions): void {
  app.get("/v1/me/navigation", async (request) => {
    const user = requireAuth(request, options.db);
    const parsed = navigationQuerySchema.safeParse(request.query ?? {});

    if (!parsed.success) {
      throw new HubServerError({
        code: "E_SCHEMA_INVALID",
        message: "Invalid navigation query.",
        retryable: false,
        statusCode: 400,
        details: { issues: parsed.error.issues },
        causeType: "navigation_query"
      });
    }

    const membership = options.db.getMembershipRole(user.user_id, parsed.data.workspace_id);
    if (!membership) {
      throw new HubServerError({
        code: "E_WORKSPACE_FORBIDDEN",
        message: "You are not a member of this workspace.",
        retryable: false,
        statusCode: 403,
        causeType: "workspace_membership"
      });
    }

    const permissions = options.db.listPermissionsForRole(membership.role_id);
    const menus = buildMenuTree(options.db.listMenusForRole(membership.role_id));

    return {
      workspace_id: parsed.data.workspace_id,
      menus,
      permissions,
      feature_flags: {}
    };
  });
}
