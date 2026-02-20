import type { FastifyRequest } from "fastify";

import type { HubDatabase } from "../db";
import { HubServerError } from "../errors";
import { hashToken } from "./tokens";

export interface AuthUser {
  user_id: string;
  email: string;
  display_name: string;
}

declare module "fastify" {
  interface FastifyRequest {
    auth_user?: AuthUser;
  }
}

export function requireAuth(request: FastifyRequest, db: HubDatabase): AuthUser {
  const header = request.headers.authorization;
  if (!header || !header.startsWith("Bearer ")) {
    throw new HubServerError({
      code: "E_AUTH_REQUIRED",
      message: "Missing bearer token.",
      retryable: false,
      statusCode: 401,
      causeType: "missing_bearer"
    });
  }

  const rawToken = header.replace(/^Bearer\s+/, "").trim();
  if (!rawToken) {
    throw new HubServerError({
      code: "E_AUTH_REQUIRED",
      message: "Missing bearer token.",
      retryable: false,
      statusCode: 401,
      causeType: "empty_bearer"
    });
  }

  const tokenHash = hashToken(rawToken);
  const record = db.getAuthTokenByHash(tokenHash);

  if (!record) {
    throw new HubServerError({
      code: "E_AUTH_INVALID",
      message: "Invalid bearer token.",
      retryable: false,
      statusCode: 401,
      causeType: "invalid_bearer"
    });
  }

  if (new Date(record.expires_at).getTime() <= Date.now()) {
    throw new HubServerError({
      code: "E_AUTH_EXPIRED",
      message: "Bearer token has expired.",
      retryable: false,
      statusCode: 401,
      causeType: "expired_bearer"
    });
  }

  if (record.user_status !== "active") {
    throw new HubServerError({
      code: "E_AUTH_INVALID",
      message: "User is disabled.",
      retryable: false,
      statusCode: 401,
      causeType: "user_disabled"
    });
  }

  db.touchAuthToken(record.token_id);

  const user: AuthUser = {
    user_id: record.user_id,
    email: record.email,
    display_name: record.display_name
  };
  request.auth_user = user;
  return user;
}
