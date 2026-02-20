import { randomUUID } from "node:crypto";

import Fastify, { type FastifyInstance } from "fastify";

import type { HubDatabase } from "./db";
import { TRACE_HEADER, errorFromUnknown } from "./errors";
import { registerBootstrapRoutes } from "./routes/bootstrap";

declare module "fastify" {
  interface FastifyRequest {
    trace_id: string;
    request_start_ms: number;
  }
}

interface CreateAppOptions {
  db: HubDatabase;
  bootstrapToken?: string;
  allowPublicSignup?: boolean;
  tokenTtlSeconds?: number;
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
    reply.status(mapped.statusCode).send({ error: mapped.error });
  });

  registerBootstrapRoutes(app, {
    db: options.db,
    bootstrapToken: options.bootstrapToken ?? "",
    allowPublicSignup: options.allowPublicSignup ?? false,
    tokenTtlSeconds: options.tokenTtlSeconds ?? 7 * 24 * 60 * 60
  });

  app.get("/v1/health", async () => ({
    ok: true,
    service: "hub-server",
    version: "0.1.0",
    ts: new Date().toISOString()
  }));

  app.get("/v1/version", async () => ({
    service: "hub-server",
    version: "0.1.0",
    protocol_version: "1.0.0"
  }));

  app.get("/healthz", async () => ({ ok: true }));

  return app;
}
