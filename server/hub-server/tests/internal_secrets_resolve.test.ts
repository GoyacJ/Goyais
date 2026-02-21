import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { HubDatabase } from "../src/db";
import { encryptApiKey } from "../src/services/secretCrypto";

const migrationsDir = path.resolve(process.cwd(), "migrations");

function makeSecretKey(): string {
  return Buffer.from("0123456789abcdef0123456789abcdef", "utf8").toString("base64");
}

describe("internal secrets resolve", () => {
  let app: Awaited<ReturnType<typeof createApp>> | undefined;
  let db: HubDatabase | undefined;

  afterEach(async () => {
    if (app) {
      await app.close();
      app = undefined;
    }
  });

  async function buildAppAndSeedSecret() {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-hub-"));
    db = new HubDatabase(path.join(tempDir, "hub.sqlite"));
    db.migrate(migrationsDir);

    const hubSecretKey = makeSecretKey();
    app = createApp({
      db,
      bootstrapToken: "bootstrap-123",
      allowPublicSignup: false,
      tokenTtlSeconds: 604800,
      hubSecretKey,
      hubRuntimeSharedSecret: "hub-runtime-secret"
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

    const bootstrapPayload = bootstrap.json() as {
      user: { user_id: string };
      workspace: { workspace_id: string };
    };
    const secretRef = "secret:test-ref";
    db.execute(
      `
      INSERT INTO secrets(secret_ref, workspace_id, kind, value_encrypted, created_by, created_at)
      VALUES(?, ?, 'api_key', ?, ?, ?)
      `,
      secretRef,
      bootstrapPayload.workspace.workspace_id,
      encryptApiKey("sk-remote-test", hubSecretKey),
      bootstrapPayload.user.user_id,
      new Date().toISOString()
    );

    return {
      workspaceId: bootstrapPayload.workspace.workspace_id,
      secretRef
    };
  }

  it("returns 401 without valid X-Hub-Auth", async () => {
    const seeded = await buildAppAndSeedSecret();

    const response = await app!.inject({
      method: "POST",
      url: "/internal/secrets/resolve",
      payload: {
        workspace_id: seeded.workspaceId,
        secret_ref: seeded.secretRef
      }
    });

    expect(response.statusCode).toBe(401);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_UNAUTHORIZED"
      }
    });
  });

  it("decrypts and returns value for valid runtime caller", async () => {
    const seeded = await buildAppAndSeedSecret();

    const response = await app!.inject({
      method: "POST",
      url: "/internal/secrets/resolve",
      headers: {
        "X-Hub-Auth": "hub-runtime-secret"
      },
      payload: {
        workspace_id: seeded.workspaceId,
        secret_ref: seeded.secretRef
      }
    });

    expect(response.statusCode).toBe(200);
    expect(response.json()).toEqual({
      value: "sk-remote-test"
    });
    expect(response.headers["x-trace-id"]).toBeTruthy();
  });
});
