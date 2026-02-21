import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { HubDatabase } from "../src/db";

const migrationsDir = path.resolve(process.cwd(), "migrations");

describe("bootstrap admin", () => {
  let app: Awaited<ReturnType<typeof createApp>> | undefined;

  afterEach(async () => {
    if (app) {
      await app.close();
      app = undefined;
    }
  });

  it("returns setup_mode=true on fresh db", async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-hub-"));
    const db = new HubDatabase(path.join(tempDir, "hub.sqlite"));
    db.migrate(migrationsDir);

    app = createApp({ db, bootstrapToken: "bootstrap-123", allowPublicSignup: false, tokenTtlSeconds: 604800 });

    const response = await app.inject({ method: "GET", url: "/v1/auth/bootstrap/status" });
    expect(response.statusCode).toBe(200);
    expect(response.json()).toMatchObject({
      setup_mode: true,
      allow_public_signup: false,
      message: "setup required"
    });
  });

  it("rejects create-admin when bootstrap token is invalid", async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-hub-"));
    const db = new HubDatabase(path.join(tempDir, "hub.sqlite"));
    db.migrate(migrationsDir);

    app = createApp({ db, bootstrapToken: "bootstrap-123", allowPublicSignup: false, tokenTtlSeconds: 604800 });

    const response = await app.inject({
      method: "POST",
      url: "/v1/auth/bootstrap/admin",
      payload: {
        bootstrap_token: "wrong",
        email: "admin@example.com",
        password: "Passw0rd!",
        display_name: "Admin"
      }
    });

    expect(response.statusCode).toBe(401);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_BOOTSTRAP_TOKEN_INVALID"
      }
    });
  });

  it("creates admin + default workspace and flips setup_mode", async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-hub-"));
    const db = new HubDatabase(path.join(tempDir, "hub.sqlite"));
    db.migrate(migrationsDir);

    app = createApp({ db, bootstrapToken: "bootstrap-123", allowPublicSignup: false, tokenTtlSeconds: 604800 });

    const createResponse = await app.inject({
      method: "POST",
      url: "/v1/auth/bootstrap/admin",
      payload: {
        bootstrap_token: "bootstrap-123",
        email: "admin@example.com",
        password: "Passw0rd!",
        display_name: "Admin"
      }
    });

    expect(createResponse.statusCode).toBe(200);
    expect(createResponse.json()).toMatchObject({
      token: expect.any(String),
      user: {
        email: "admin@example.com",
        display_name: "Admin"
      },
      workspace: {
        name: "Default",
        slug: "default"
      }
    });

    const statusResponse = await app.inject({ method: "GET", url: "/v1/auth/bootstrap/status" });
    expect(statusResponse.statusCode).toBe(200);
    expect(statusResponse.json()).toMatchObject({
      setup_mode: false,
      message: "ok"
    });
  });

  it("prevents bootstrap once setup completed", async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-hub-"));
    const db = new HubDatabase(path.join(tempDir, "hub.sqlite"));
    db.migrate(migrationsDir);

    app = createApp({ db, bootstrapToken: "bootstrap-123", allowPublicSignup: false, tokenTtlSeconds: 604800 });

    const payload = {
      bootstrap_token: "bootstrap-123",
      email: "admin@example.com",
      password: "Passw0rd!",
      display_name: "Admin"
    };

    const first = await app.inject({ method: "POST", url: "/v1/auth/bootstrap/admin", payload });
    expect(first.statusCode).toBe(200);

    const second = await app.inject({ method: "POST", url: "/v1/auth/bootstrap/admin", payload });
    expect(second.statusCode).toBe(409);
    expect(second.json()).toMatchObject({
      error: {
        code: "E_SETUP_COMPLETED"
      }
    });
  });
});
