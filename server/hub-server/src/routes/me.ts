import type { FastifyInstance } from "fastify";

import { requireAuth } from "../auth/bearer";
import type { HubDatabase } from "../db";

interface RegisterMeRoutesOptions {
  db: HubDatabase;
}

export function registerMeRoutes(app: FastifyInstance, options: RegisterMeRoutesOptions): void {
  app.get("/v1/me", async (request) => {
    const user = requireAuth(request, options.db);
    const memberships = options.db.listMemberships(user.user_id);

    return {
      user,
      memberships
    };
  });
}
