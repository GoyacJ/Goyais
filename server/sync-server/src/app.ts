import { randomUUID } from "node:crypto";

import Fastify, { type FastifyInstance } from "fastify";

import type { SyncDatabase } from "./db";
import { TRACE_HEADER, errorFromUnknown } from "./errors";
import { createMetrics, metricsSnapshot } from "./metrics";
import { registerPullRoute } from "./routes/pull";
import { registerPushRoute } from "./routes/push";

declare module "fastify" {
  interface FastifyRequest {
    trace_id: string;
    request_start_ms: number;
  }
}

interface CreateAppOptions {
  db: SyncDatabase;
  token: string;
}

function normalizeTraceHeader(value: unknown): string {
  if (typeof value === "string" && value.trim().length > 0) {
    return value.trim();
  }
  if (Array.isArray(value) && value.length > 0 && typeof value[0] === "string" && value[0].trim().length > 0) {
    return value[0].trim();
  }
  return randomUUID();
}

export function createApp(options: CreateAppOptions): FastifyInstance {
  const metrics = createMetrics();
  const app = Fastify({ logger: true });

  app.addHook("onRequest", async (request) => {
    request.trace_id = normalizeTraceHeader(request.headers["x-trace-id"]);
    request.request_start_ms = Date.now();
  });

  app.addHook("onSend", async (request, reply, payload) => {
    reply.header(TRACE_HEADER, request.trace_id);
    return payload;
  });

  app.addHook("onResponse", async (request, reply) => {
    const latencyMs = Date.now() - request.request_start_ms;
    app.log.info(
      {
        trace_id: request.trace_id,
        route: request.routeOptions.url ?? request.url,
        status: reply.statusCode,
        latency_ms: latencyMs
      },
      "request_complete"
    );
  });

  app.setErrorHandler((error, request, reply) => {
    const mapped = errorFromUnknown(error, request.trace_id || randomUUID());
    if (mapped.error.code === "E_SYNC_AUTH") {
      metrics.auth_fail_total += 1;
    }
    reply.status(mapped.statusCode).send({ error: mapped.error });
  });

  registerPushRoute(app, options.db, options.token, metrics);
  registerPullRoute(app, options.db, options.token, metrics);

  app.get("/v1/health", async () => ({ ok: true, runtime_status: "ok" }));
  app.get("/v1/version", async () => ({ protocol_version: "2.0.0", runtime_version: "0.2.0" }));
  app.get("/v1/metrics", async () => metricsSnapshot(options.db, metrics));

  app.get("/healthz", async () => ({ ok: true }));

  return app;
}
