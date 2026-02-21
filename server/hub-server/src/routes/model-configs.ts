import { z } from "zod";

import type { FastifyInstance } from "fastify";

import {
  requireDomainAuth,
  requirePermission,
  requireWorkspaceIdQuery,
  requireWorkspaceMember
} from "../auth/workspace-rbac";
import type { HubDatabase } from "../db";
import { HubServerError } from "../errors";
import { encryptApiKey } from "../services/secretCrypto";

const providerSchema = z.enum([
  "deepseek",
  "minimax_cn",
  "minimax_intl",
  "zhipu",
  "qwen",
  "doubao",
  "openai",
  "anthropic",
  "google",
  "custom"
]);

const createModelConfigSchema = z.object({
  provider: providerSchema,
  model: z.string().min(1),
  base_url: z.string().min(1).nullable().optional(),
  temperature: z.number().finite().default(0),
  max_tokens: z.number().int().positive().nullable().optional(),
  api_key: z.string().min(1)
});

const updateModelConfigSchema = z
  .object({
    provider: providerSchema.optional(),
    model: z.string().min(1).optional(),
    base_url: z.string().min(1).nullable().optional(),
    temperature: z.number().finite().optional(),
    max_tokens: z.number().int().positive().nullable().optional(),
    api_key: z.string().min(1).optional()
  })
  .refine((data) => Object.keys(data).length > 0, {
    message: "At least one field must be provided."
  });

const modelConfigParamsSchema = z.object({
  model_config_id: z.string().min(1)
});

interface RegisterModelConfigRoutesOptions {
  db: HubDatabase;
  hubSecretKey: string;
}

export function registerModelConfigRoutes(app: FastifyInstance, options: RegisterModelConfigRoutesOptions): void {
  app.get("/v1/model-configs", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "modelconfig:read");

    const modelConfigs = options.db.listModelConfigs(workspaceId);
    return {
      model_configs: modelConfigs
    };
  });

  app.post("/v1/model-configs", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "modelconfig:manage");

    const parsed = createModelConfigSchema.safeParse(request.body);
    if (!parsed.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid model config payload.",
        retryable: false,
        statusCode: 400,
        details: { issues: parsed.error.issues },
        causeType: "model_config_payload"
      });
    }

    const encryptedApiKey = encryptApiKey(parsed.data.api_key, options.hubSecretKey);
    const modelConfig = options.db.createModelConfigWithSecret({
      workspaceId,
      provider: parsed.data.provider,
      model: parsed.data.model.trim(),
      baseUrl: parsed.data.base_url ?? null,
      temperature: parsed.data.temperature ?? 0,
      maxTokens: parsed.data.max_tokens ?? null,
      createdBy: user.user_id,
      encryptedApiKey
    });

    return {
      model_config: modelConfig
    };
  });

  app.put("/v1/model-configs/:model_config_id", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "modelconfig:manage");

    const paramsParsed = modelConfigParamsSchema.safeParse(request.params);
    if (!paramsParsed.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid model config route params.",
        retryable: false,
        statusCode: 400,
        details: { issues: paramsParsed.error.issues },
        causeType: "model_config_params"
      });
    }

    const bodyParsed = updateModelConfigSchema.safeParse(request.body);
    if (!bodyParsed.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid model config payload.",
        retryable: false,
        statusCode: 400,
        details: { issues: bodyParsed.error.issues },
        causeType: "model_config_payload"
      });
    }

    const encryptedApiKey = bodyParsed.data.api_key
      ? encryptApiKey(bodyParsed.data.api_key, options.hubSecretKey)
      : undefined;
    const updated = options.db.updateModelConfigWithOptionalSecretRotation({
      workspaceId,
      modelConfigId: paramsParsed.data.model_config_id,
      provider: bodyParsed.data.provider,
      model: bodyParsed.data.model?.trim(),
      baseUrl: bodyParsed.data.base_url,
      temperature: bodyParsed.data.temperature,
      maxTokens: bodyParsed.data.max_tokens,
      createdBy: user.user_id,
      encryptedApiKey
    });

    if (!updated) {
      throw new HubServerError({
        code: "E_NOT_FOUND",
        message: "Model config not found.",
        retryable: false,
        statusCode: 404,
        causeType: "model_config_lookup"
      });
    }

    return {
      model_config: updated
    };
  });

  app.delete("/v1/model-configs/:model_config_id", async (request) => {
    const user = requireDomainAuth(request, options.db);
    const workspaceId = requireWorkspaceIdQuery(request);
    const membership = requireWorkspaceMember(request, options.db, user, workspaceId);
    requirePermission(options.db, membership.role_id, "modelconfig:manage");

    const paramsParsed = modelConfigParamsSchema.safeParse(request.params);
    if (!paramsParsed.success) {
      throw new HubServerError({
        code: "E_VALIDATION",
        message: "Invalid model config route params.",
        retryable: false,
        statusCode: 400,
        details: { issues: paramsParsed.error.issues },
        causeType: "model_config_params"
      });
    }

    const deleted = options.db.deleteModelConfigAndSecret(workspaceId, paramsParsed.data.model_config_id);
    if (!deleted) {
      throw new HubServerError({
        code: "E_NOT_FOUND",
        message: "Model config not found.",
        retryable: false,
        statusCode: 404,
        causeType: "model_config_lookup"
      });
    }

    return {
      ok: true
    };
  });
}
