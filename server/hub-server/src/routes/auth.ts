import { z } from "zod";

import type { FastifyInstance } from "fastify";

import { verifyPassword } from "../auth/password";
import { issueOpaqueToken } from "../auth/tokens";
import type { HubDatabase } from "../db";
import { HubServerError } from "../errors";

const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(1)
});

interface RegisterAuthRoutesOptions {
  db: HubDatabase;
  tokenTtlSeconds: number;
}

export function registerAuthRoutes(app: FastifyInstance, options: RegisterAuthRoutesOptions): void {
  app.post("/v1/auth/login", async (request) => {
    const parsed = loginSchema.safeParse(request.body);
    if (!parsed.success) {
      throw new HubServerError({
        code: "E_SCHEMA_INVALID",
        message: "Invalid login payload.",
        retryable: false,
        statusCode: 400,
        details: { issues: parsed.error.issues },
        causeType: "login_payload"
      });
    }

    const email = parsed.data.email.toLowerCase();
    const user = options.db.getUserByEmail(email);

    if (!user || !verifyPassword(parsed.data.password, user.password_hash)) {
      throw new HubServerError({
        code: "E_AUTH_INVALID",
        message: "Invalid email or password.",
        retryable: false,
        statusCode: 401,
        causeType: "invalid_login"
      });
    }

    if (user.status !== "active") {
      throw new HubServerError({
        code: "E_AUTH_INVALID",
        message: "User is disabled.",
        retryable: false,
        statusCode: 401,
        causeType: "user_disabled"
      });
    }

    const issued = issueOpaqueToken(options.tokenTtlSeconds);
    options.db.createAuthToken({
      tokenId: issued.tokenId,
      tokenHash: issued.tokenHash,
      userId: user.user_id,
      expiresAt: issued.expiresAt,
      createdAt: issued.createdAt
    });

    return {
      token: issued.token,
      user: {
        user_id: user.user_id,
        email: user.email,
        display_name: user.display_name
      }
    };
  });
}
