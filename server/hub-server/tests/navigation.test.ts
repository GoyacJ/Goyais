import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { HubDatabase } from "../src/db";

const migrationsDir = path.resolve(process.cwd(), "migrations");

describe("navigation", () => {
  let app: Awaited<ReturnType<typeof createApp>> | undefined;

  afterEach(async () => {
    if (app) {
      await app.close();
      app = undefined;
    }
  });

  async function bootstrapAndLogin() {
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

    const login = await app.inject({
      method: "POST",
      url: "/v1/auth/login",
      payload: {
        email: "admin@example.com",
        password: "Passw0rd!"
      }
    });
    expect(login.statusCode).toBe(200);

    const loginPayload = login.json() as { token: string };

    const workspaces = await app.inject({
      method: "GET",
      url: "/v1/workspaces",
      headers: {
        Authorization: `Bearer ${loginPayload.token}`
      }
    });

    const workspacePayload = workspaces.json() as {
      workspaces: Array<{ workspace_id: string }>;
    };

    return {
      token: loginPayload.token,
      workspaceId: workspacePayload.workspaces[0].workspace_id
    };
  }

  it("returns menu tree and permissions for active membership", async () => {
    const auth = await bootstrapAndLogin();

    const response = await app!.inject({
      method: "GET",
      url: `/v1/me/navigation?workspace_id=${encodeURIComponent(auth.workspaceId)}`,
      headers: {
        Authorization: `Bearer ${auth.token}`
      }
    });

    expect(response.statusCode).toBe(200);
    expect(response.json()).toMatchObject({
      workspace_id: auth.workspaceId,
      permissions: expect.arrayContaining(["workspace:manage", "project:read", "run:create"]),
      menus: expect.arrayContaining([
        expect.objectContaining({ menu_id: "nav_projects", route: "/projects", i18n_key: "nav.projects" }),
        expect.objectContaining({ menu_id: "nav_run", route: "/run", i18n_key: "nav.run" })
      ]),
      feature_flags: {}
    });
  });

  it("rejects navigation request for non-member workspace", async () => {
    const auth = await bootstrapAndLogin();

    const response = await app!.inject({
      method: "GET",
      url: "/v1/me/navigation?workspace_id=ws-not-member",
      headers: {
        Authorization: `Bearer ${auth.token}`
      }
    });

    expect(response.statusCode).toBe(403);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_WORKSPACE_FORBIDDEN"
      }
    });
  });
});
