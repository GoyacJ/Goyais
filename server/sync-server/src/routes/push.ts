import { z } from "zod";

import type { FastifyInstance } from "fastify";

import { assertToken } from "../auth";
import type { SyncDatabase } from "../db";
import { SyncServerError } from "../errors";
import type { SyncMetrics } from "../metrics";

const pushSchema = z.object({
  device_id: z.string().min(1),
  since_global_seq: z.number().int().nonnegative(),
  events: z.array(
    z.object({
      protocol_version: z.literal("2.0.0"),
      trace_id: z.string().min(1),
      event_id: z.string().min(1),
      run_id: z.string().min(1),
      seq: z.number().int().positive(),
      ts: z.string().min(1),
      type: z.enum(["plan", "tool_call", "tool_result", "patch", "error", "done"]),
      payload: z.record(z.any())
    })
  ),
  artifacts_meta: z.array(z.record(z.any()))
});

export function registerPushRoute(app: FastifyInstance, db: SyncDatabase, token: string, metrics: SyncMetrics) {
  app.post("/v1/sync/push", async (request, reply) => {
    metrics.push_requests_total += 1;
    assertToken(request, token);

    const parsed = pushSchema.safeParse(request.body);
    if (!parsed.success) {
      throw new SyncServerError({
        code: "E_SCHEMA_INVALID",
        message: "Invalid push payload.",
        retryable: false,
        statusCode: 400,
        details: { issues: parsed.error.issues },
        causeType: "push_schema"
      });
    }

    const result = db.push(parsed.data);
    request.log.info({
      trace_id: request.trace_id,
      inserted_count: result.inserted,
      max_server_seq: result.max_server_seq,
      route: "/v1/sync/push"
    });
    return reply.send(result);
  });
}
