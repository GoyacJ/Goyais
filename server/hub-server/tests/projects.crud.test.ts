import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { HubDatabase } from "../src/db";

const migrationsDir = path.resolve(process.cwd(), "migrations");

describe("projects crud", () => {
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
    const me = await app.inject({
      method: "GET",
      url: "/v1/me",
      headers: {
        Authorization: `Bearer ${token}`
      }
    });
    const userId = (me.json() as { user: { user_id: string } }).user.user_id;

    const workspaces = await app.inject({
      method: "GET",
      url: "/v1/workspaces",
      headers: {
        Authorization: `Bearer ${token}`
      }
    });
    const workspaceId = (workspaces.json() as { workspaces: Array<{ workspace_id: string }> }).workspaces[0].workspace_id;

    return { token, workspaceId, userId };
  }

  it("creates and deletes project when role has project:write", async () => {
    const { token, workspaceId } = await buildAppAndLogin();

    const createResponse = await app!.inject({
      method: "POST",
      url: `/v1/projects?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      },
      payload: {
        name: "Remote Project A",
        root_uri: "repo://sample/a"
      }
    });

    expect(createResponse.statusCode).toBe(200);
    const projectId = (createResponse.json() as { project: { project_id: string } }).project.project_id;

    const listAfterCreate = await app!.inject({
      method: "GET",
      url: `/v1/projects?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      }
    });
    expect(listAfterCreate.statusCode).toBe(200);
    expect(listAfterCreate.json()).toMatchObject({
      projects: [
        expect.objectContaining({
          project_id: projectId,
          name: "Remote Project A",
          root_uri: "repo://sample/a"
        })
      ]
    });

    const deleteResponse = await app!.inject({
      method: "DELETE",
      url: `/v1/projects/${projectId}?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      }
    });

    expect(deleteResponse.statusCode).toBe(200);
    expect(deleteResponse.json()).toMatchObject({ ok: true });
  });

  it("returns 403 for create/delete when role misses project:write", async () => {
    const { token, workspaceId, userId } = await buildAppAndLogin();
    const membership = db!.getMembershipRole(userId, workspaceId);
    expect(membership).toBeTruthy();

    db!.execute(
      "DELETE FROM role_permissions WHERE role_id = ? AND perm_key = 'project:write'",
      membership!.role_id
    );

    const createResponse = await app!.inject({
      method: "POST",
      url: `/v1/projects?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      },
      payload: {
        name: "No Write",
        root_uri: "repo://sample/nowrite"
      }
    });
    expect(createResponse.statusCode).toBe(403);
    expect(createResponse.json()).toMatchObject({
      error: {
        code: "E_FORBIDDEN"
      }
    });

    db!.execute(
      `
      INSERT INTO projects(project_id, workspace_id, name, root_uri, created_by, created_at, updated_at)
      VALUES('p1', ?, 'P1', 'repo://sample/p1', ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))
      `,
      workspaceId,
      userId
    );

    const deleteResponse = await app!.inject({
      method: "DELETE",
      url: `/v1/projects/p1?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      }
    });
    expect(deleteResponse.statusCode).toBe(403);
    expect(deleteResponse.json()).toMatchObject({
      error: {
        code: "E_FORBIDDEN"
      }
    });
  });
});
