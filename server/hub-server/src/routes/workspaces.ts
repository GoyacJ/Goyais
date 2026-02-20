import type { FastifyInstance } from "fastify";

import { requireAuth } from "../auth/bearer";
import type { HubDatabase } from "../db";

interface RegisterWorkspaceRoutesOptions {
  db: HubDatabase;
}

export function registerWorkspaceRoutes(app: FastifyInstance, options: RegisterWorkspaceRoutesOptions): void {
  app.get("/v1/workspaces", async (request) => {
    const user = requireAuth(request, options.db);
    const workspaces = options.db.listWorkspacesForUser(user.user_id);

    return {
      workspaces
    };
  });
}
