import { z } from "zod";

import type { FastifyInstance } from "fastify";

import { hashPassword } from "../auth/password";
import { issueOpaqueToken } from "../auth/tokens";
import type { HubDatabase } from "../db";
import { HubServerError } from "../errors";

const bootstrapCreateSchema = z.object({
  bootstrap_token: z.string().min(1),
  email: z.string().email(),
  password: z.string().min(8),
  display_name: z.string().min(1)
});

interface RegisterBootstrapOptions {
  db: HubDatabase;
  bootstrapToken: string;
  allowPublicSignup: boolean;
  tokenTtlSeconds: number;
}

export function registerBootstrapRoutes(app: FastifyInstance, options: RegisterBootstrapOptions): void {
  app.get("/v1/auth/bootstrap/status", async () => {
    const status = options.db.getSetupStatus();

    return {
      setup_mode: status.setupMode,
      allow_public_signup: options.allowPublicSignup,
      message: status.setupMode ? "setup required" : "ok"
    };
  });

  app.post("/v1/auth/bootstrap/admin", async (request) => {
    const parsed = bootstrapCreateSchema.safeParse(request.body);
    if (!parsed.success) {
      throw new HubServerError({
        code: "E_SCHEMA_INVALID",
        message: "Invalid bootstrap payload.",
        retryable: false,
        statusCode: 400,
        details: { issues: parsed.error.issues },
        causeType: "bootstrap_payload"
      });
    }

    const setup = options.db.getSetupStatus();
    if (!setup.setupMode) {
      throw new HubServerError({
        code: "E_SETUP_COMPLETED",
        message: "Bootstrap has already been completed.",
        retryable: false,
        statusCode: 409,
        causeType: "setup_completed"
      });
    }

    if (!options.bootstrapToken || parsed.data.bootstrap_token !== options.bootstrapToken) {
      throw new HubServerError({
        code: "E_BOOTSTRAP_TOKEN_INVALID",
        message: "Invalid bootstrap token.",
        retryable: false,
        statusCode: 401,
        causeType: "bootstrap_token"
      });
    }

    const passwordHash = hashPassword(parsed.data.password);
    const issued = issueOpaqueToken(options.tokenTtlSeconds);
    const result = options.db.createBootstrapAdmin({
      email: parsed.data.email.toLowerCase(),
      passwordHash,
      displayName: parsed.data.display_name,
      tokenId: issued.tokenId,
      tokenHash: issued.tokenHash,
      tokenCreatedAt: issued.createdAt,
      tokenExpiresAt: issued.expiresAt
    });

    return {
      token: issued.token,
      user: result.user,
      workspace: result.workspace
    };
  });
}
