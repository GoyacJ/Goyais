import type { FastifyRequest } from "fastify";

import { SyncServerError } from "./errors";

export function assertToken(request: FastifyRequest, expectedToken: string): void {
  const header = request.headers.authorization;
  if (!header || !header.startsWith("Bearer ")) {
    throw new SyncServerError({
      code: "E_SYNC_AUTH",
      message: "Missing bearer token.",
      retryable: false,
      statusCode: 401,
      causeType: "missing_bearer"
    });
  }

  const token = header.replace(/^Bearer\s+/, "").trim();
  if (token !== expectedToken) {
    throw new SyncServerError({
      code: "E_SYNC_AUTH",
      message: "Invalid bearer token.",
      retryable: false,
      statusCode: 401,
      causeType: "invalid_bearer"
    });
  }
}
