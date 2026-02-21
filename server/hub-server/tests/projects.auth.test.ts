import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { HubDatabase } from "../src/db";

const migrationsDir = path.resolve(process.cwd(), "migrations");

describe("projects auth and workspace rbac", () => {
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

  it("returns 401 E_UNAUTHORIZED when not logged in", async () => {
    const { workspaceId } = await buildAppAndLogin();

    const response = await app!.inject({
      method: "GET",
      url: `/v1/projects?workspace_id=${workspaceId}`
    });

    expect(response.statusCode).toBe(401);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_UNAUTHORIZED"
      }
    });
  });

  it("returns 403 E_FORBIDDEN for non-member workspace", async () => {
    const { token } = await buildAppAndLogin();

    const response = await app!.inject({
      method: "GET",
      url: "/v1/projects?workspace_id=ws-not-member",
      headers: {
        Authorization: `Bearer ${token}`
      }
    });

    expect(response.statusCode).toBe(403);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_FORBIDDEN"
      }
    });
  });

  it("returns 403 E_FORBIDDEN without project:read permission", async () => {
    const { token, workspaceId } = await buildAppAndLogin();

    const me = await app!.inject({
      method: "GET",
      url: "/v1/me",
      headers: {
        Authorization: `Bearer ${token}`
      }
    });
    const userId = (me.json() as { user: { user_id: string } }).user.user_id;
    const membership = db!.getMembershipRole(userId, workspaceId);
    expect(membership).toBeTruthy();

    db!.execute(
      "DELETE FROM role_permissions WHERE role_id = ? AND perm_key = 'project:read'",
      membership!.role_id
    );

    const response = await app!.inject({
      method: "GET",
      url: `/v1/projects?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      }
    });

    expect(response.statusCode).toBe(403);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_FORBIDDEN"
      }
    });
  });

  it("returns 200 when user has project:read", async () => {
    const { token, workspaceId } = await buildAppAndLogin();

    const response = await app!.inject({
      method: "GET",
      url: `/v1/projects?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      }
    });

    expect(response.statusCode).toBe(200);
    expect(response.json()).toMatchObject({
      projects: []
    });
  });
});
