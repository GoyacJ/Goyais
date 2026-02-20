import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { HubDatabase } from "../src/db";

const migrationsDir = path.resolve(process.cwd(), "migrations");
const TEST_SECRET_KEY = Buffer.alloc(32, 7).toString("base64");

describe("model-configs crud + secret storage", () => {
  let app: Awaited<ReturnType<typeof createApp>> | undefined;
  let db: HubDatabase | undefined;

  afterEach(async () => {
    if (app) {
      await app.close();
      app = undefined;
    }
  });

  async function buildAppAndLogin() {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-hub-"));
    db = new HubDatabase(path.join(tempDir, "hub.sqlite"));
    db.migrate(migrationsDir);

    app = createApp({
      db,
      bootstrapToken: "bootstrap-123",
      hubSecretKey: TEST_SECRET_KEY,
      allowPublicSignup: false,
      tokenTtlSeconds: 604800
    });

    const bootstrap = await app.inject({
      method: "POST",
      url: "/v1/auth/bootstrap/admin",
      payload: {
        bootstrap_token: "bootstrap-123",
        email: "admin@example.com",
        password: "Passw0rd!",
        display_name: "Admin"
      }
    });
    expect(bootstrap.statusCode).toBe(200);

    const login = await app.inject({
      method: "POST",
      url: "/v1/auth/login",
      payload: {
        email: "admin@example.com",
        password: "Passw0rd!"
      }
    });
    expect(login.statusCode).toBe(200);

    const token = (login.json() as { token: string }).token;
    const workspaces = await app.inject({
      method: "GET",
      url: "/v1/workspaces",
      headers: {
        Authorization: `Bearer ${token}`
      }
    });
    const workspaceId = (workspaces.json() as { workspaces: Array<{ workspace_id: string }> }).workspaces[0].workspace_id;

    return { token, workspaceId };
  }

  it("creates model config and stores encrypted secret without returning api_key", async () => {
    const { token, workspaceId } = await buildAppAndLogin();

    const response = await app!.inject({
      method: "POST",
      url: `/v1/model-configs?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      },
      payload: {
        provider: "openai",
        model: "gpt-4.1-mini",
        temperature: 0,
        max_tokens: 2048,
        api_key: "sk-remote-secret"
      }
    });

    expect(response.statusCode).toBe(200);
    const payload = response.json() as {
      model_config: {
        model_config_id: string;
        secret_ref: string;
        api_key?: string;
      };
    };
    expect(payload.model_config.secret_ref).toMatch(/^secret:/);
    expect(payload.model_config.api_key).toBeUndefined();

    const secretCount = db!.scalar<number>(
      "SELECT COUNT(*) FROM secrets WHERE workspace_id = ?",
      workspaceId
    );
    expect(secretCount).toBe(1);

    const encrypted = db!.scalar<string>(
      "SELECT value_encrypted FROM secrets WHERE secret_ref = ?",
      payload.model_config.secret_ref
    );
    expect(encrypted).toContain("enc:v1:");
    expect(encrypted.includes("sk-remote-secret")).toBe(false);
  });

  it("rotates secret_ref when updating api_key", async () => {
    const { token, workspaceId } = await buildAppAndLogin();

    const created = await app!.inject({
      method: "POST",
      url: `/v1/model-configs?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      },
      payload: {
        provider: "openai",
        model: "gpt-4.1-mini",
        temperature: 0,
        api_key: "sk-old"
      }
    });
    expect(created.statusCode).toBe(200);

    const createdPayload = created.json() as {
      model_config: {
        model_config_id: string;
        secret_ref: string;
      };
    };

    const updated = await app!.inject({
      method: "PUT",
      url: `/v1/model-configs/${createdPayload.model_config.model_config_id}?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      },
      payload: {
        model: "gpt-4.1",
        api_key: "sk-new"
      }
    });
    expect(updated.statusCode).toBe(200);

    const updatedPayload = updated.json() as {
      model_config: {
        secret_ref: string;
      };
    };
    expect(updatedPayload.model_config.secret_ref).not.toBe(createdPayload.model_config.secret_ref);

    const oldSecretCount = db!.scalar<number>(
      "SELECT COUNT(*) FROM secrets WHERE secret_ref = ?",
      createdPayload.model_config.secret_ref
    );
    expect(oldSecretCount).toBe(0);
  });

  it("does not expose api_key in list endpoint", async () => {
    const { token, workspaceId } = await buildAppAndLogin();

    const created = await app!.inject({
      method: "POST",
      url: `/v1/model-configs?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      },
      payload: {
        provider: "anthropic",
        model: "claude-3-5-sonnet",
        temperature: 0.1,
        api_key: "sk-hidden"
      }
    });
    expect(created.statusCode).toBe(200);

    const list = await app!.inject({
      method: "GET",
      url: `/v1/model-configs?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      }
    });
    expect(list.statusCode).toBe(200);

    const payload = list.json() as {
      model_configs: Array<Record<string, unknown>>;
    };
    expect(payload.model_configs.length).toBeGreaterThan(0);
    for (const item of payload.model_configs) {
      expect(item).not.toHaveProperty("api_key");
    }
  });
});
