import { z } from "zod";

import type { FastifyInstance } from "fastify";

import type { HubDatabase } from "../db";
import { HubServerError } from "../errors";
import { decryptApiKey } from "../services/secretCrypto";

const resolveSecretSchema = z.object({
  workspace_id: z.string().min(1),
  secret_ref: z.string().min(1)
});

interface RegisterInternalSecretsRoutesOptions {
  db: HubDatabase;
  hubSecretKey: string;
  runtimeSharedSecret: string;
}

function normalizeHeaderValue(value: unknown): string {
  if (typeof value === "string") {
    return value.trim();
  }
  if (Array.isArray(value) && typeof value[0] === "string") {
    return value[0].trim();
  }
  return "";
}

export function registerInternalSecretsRoutes(
  app: FastifyInstance,
  options: RegisterInternalSecretsRoutesOptions
): void {
  app.post("/internal/secrets/resolve", async (request) => {
    const expectedSecret = options.runtimeSharedSecret.trim();
    const providedSecret = normalizeHeaderValue(request.headers["x-hub-auth"]);

    if (!expectedSecret || !providedSecret || providedSecret !== expectedSecret) {
      throw new HubServerError({
        code: "E_UNAUTHORIZED",
        message: "Unauthorized.",
        retryable: false,
        statusCode: 401,
        causeType: "internal_secret_auth"
      });
    }

    const parsed = resolveSecretSchema.safeParse(request.body);
    if (!parsed.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid secret resolve payload.",
        retryable: false,
        statusCode: 400,
        details: { issues: parsed.error.issues },
        causeType: "internal_secret_payload"
      });
    }

    const secret = options.db.getSecretByRef(parsed.data.workspace_id, parsed.data.secret_ref);
    if (!secret) {
      throw new HubServerError({
        code: "E_NOT_FOUND",
        message: "Secret not found.",
        retryable: false,
        statusCode: 404,
        causeType: "internal_secret_lookup"
      });
    }

    const value = decryptApiKey(secret.value_encrypted, options.hubSecretKey);
    return {
      value
    };
  });
}
