import { z } from "zod";

import type { FastifyInstance } from "fastify";

import { assertToken } from "../auth";
import type { SyncDatabase } from "../db";

const pushSchema = z.object({
  device_id: z.string().min(1),
  since_global_seq: z.number().int().nonnegative(),
  events: z.array(
    z.object({
      protocol_version: z.literal("1.0.0"),
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

export function registerPushRoute(app: FastifyInstance, db: SyncDatabase, token: string) {
  app.post("/v1/sync/push", async (request, reply) => {
    try {
      assertToken(request, token);
    } catch (error) {
      return reply.status(401).send({ error: (error as Error).message });
    }

    const parsed = pushSchema.safeParse(request.body);
    if (!parsed.success) {
      return reply.status(400).send({ error: parsed.error.format() });
    }

    const result = db.push(parsed.data);
    return reply.send(result);
  });
}
