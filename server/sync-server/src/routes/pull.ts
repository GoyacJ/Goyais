import type { FastifyInstance } from "fastify";

import { assertToken } from "../auth";
import type { SyncDatabase } from "../db";
import { SyncServerError } from "../errors";
import type { SyncMetrics } from "../metrics";

export function registerPullRoute(app: FastifyInstance, db: SyncDatabase, token: string, metrics: SyncMetrics) {
  app.get("/v1/sync/pull", async (request, reply) => {
    metrics.pull_requests_total += 1;
    assertToken(request, token);

    const sinceSeq = Number((request.query as Record<string, string>).since_server_seq ?? "0");
    if (!Number.isFinite(sinceSeq) || sinceSeq < 0) {
      throw new SyncServerError({
        code: "E_SCHEMA_INVALID",
        message: "Invalid since_server_seq.",
        retryable: false,
        statusCode: 400,
        causeType: "pull_query"
      });
    }

    const result = db.pull(Number.isFinite(sinceSeq) ? sinceSeq : 0);
    request.log.info({
      trace_id: request.trace_id,
      route: "/v1/sync/pull",
      max_server_seq: result.max_server_seq,
      events_count: result.events.length
    });
    return reply.send(result);
  });
}
