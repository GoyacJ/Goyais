import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { HubDatabase } from "../src/db";

const migrationsDir = path.resolve(process.cwd(), "migrations");

describe("auth and identity", () => {
  let app: Awaited<ReturnType<typeof createApp>> | undefined;

  afterEach(async () => {
    if (app) {
      await app.close();
      app = undefined;
    }
  });

  async function buildBootstrappedApp() {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-hub-"));
    const db = new HubDatabase(path.join(tempDir, "hub.sqlite"));
    db.migrate(migrationsDir);

    app = createApp({ db, bootstrapToken: "bootstrap-123", allowPublicSignup: false, tokenTtlSeconds: 604800 });

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
    return app;
  }

  it("logs in with email/password and returns token", async () => {
    const readyApp = await buildBootstrappedApp();

    const response = await readyApp.inject({
      method: "POST",
      url: "/v1/auth/login",
      payload: {
        email: "admin@example.com",
        password: "Passw0rd!"
      }
    });

    expect(response.statusCode).toBe(200);
    expect(response.json()).toMatchObject({
      token: expect.any(String),
      user: {
        email: "admin@example.com",
        display_name: "Admin"
      }
    });
  });

  it("rejects login when password is incorrect", async () => {
    const readyApp = await buildBootstrappedApp();

    const response = await readyApp.inject({
      method: "POST",
      url: "/v1/auth/login",
      payload: {
        email: "admin@example.com",
        password: "wrong-password"
      }
    });

    expect(response.statusCode).toBe(401);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_AUTH_INVALID"
      }
    });
  });

  it("returns /v1/me with memberships for valid bearer token", async () => {
    const readyApp = await buildBootstrappedApp();

    const login = await readyApp.inject({
      method: "POST",
      url: "/v1/auth/login",
      payload: {
        email: "admin@example.com",
        password: "Passw0rd!"
      }
    });

    const loginPayload = login.json() as { token: string };

    const me = await readyApp.inject({
      method: "GET",
      url: "/v1/me",
      headers: {
        Authorization: `Bearer ${loginPayload.token}`
      }
    });

    expect(me.statusCode).toBe(200);
    expect(me.json()).toMatchObject({
      user: {
        email: "admin@example.com",
        display_name: "Admin"
      },
      memberships: [
        {
          workspace_name: "Default",
          workspace_slug: "default",
          role_name: "Owner"
        }
      ]
    });
  });

  it("returns /v1/workspaces for valid bearer token", async () => {
    const readyApp = await buildBootstrappedApp();

    const login = await readyApp.inject({
      method: "POST",
      url: "/v1/auth/login",
      payload: {
        email: "admin@example.com",
        password: "Passw0rd!"
      }
    });

    const loginPayload = login.json() as { token: string };

    const response = await readyApp.inject({
      method: "GET",
      url: "/v1/workspaces",
      headers: {
        Authorization: `Bearer ${loginPayload.token}`
      }
    });

    expect(response.statusCode).toBe(200);
    expect(response.json()).toMatchObject({
      workspaces: [
        {
          name: "Default",
          slug: "default",
          role_name: "Owner"
        }
      ]
    });
  });

  it("rejects /v1/me without bearer token", async () => {
    const readyApp = await buildBootstrappedApp();

    const response = await readyApp.inject({
      method: "GET",
      url: "/v1/me"
    });

    expect(response.statusCode).toBe(401);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_AUTH_REQUIRED"
      }
    });
  });
});
