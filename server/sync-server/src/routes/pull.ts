import type { FastifyInstance } from "fastify";

import { assertToken } from "../auth";
import type { SyncDatabase } from "../db";

export function registerPullRoute(app: FastifyInstance, db: SyncDatabase, token: string) {
  app.get("/v1/sync/pull", async (request, reply) => {
    try {
      assertToken(request, token);
    } catch (error) {
      return reply.status(401).send({ error: (error as Error).message });
    }

    const sinceSeq = Number((request.query as Record<string, string>).since_server_seq ?? "0");
    const result = db.pull(Number.isFinite(sinceSeq) ? sinceSeq : 0);
    return reply.send(result);
  });
}
