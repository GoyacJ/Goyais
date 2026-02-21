import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { HubDatabase } from "../src/db";

const migrationsDir = path.resolve(process.cwd(), "migrations");

describe("runtime registry admin endpoint", () => {
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
    expect(me.statusCode).toBe(200);
    const userId = (me.json() as { user: { user_id: string } }).user.user_id;

    const workspaces = await app.inject({
      method: "GET",
      url: "/v1/workspaces",
      headers: {
        Authorization: `Bearer ${token}`
      }
    });
    expect(workspaces.statusCode).toBe(200);
    const workspaceId = (workspaces.json() as { workspaces: Array<{ workspace_id: string }> }).workspaces[0].workspace_id;

    return { token, userId, workspaceId };
  }

  it("registers runtime base url for workspace manager", async () => {
    const { token, workspaceId } = await buildAppAndLogin();

    const response = await app!.inject({
      method: "POST",
      url: `/v1/admin/workspaces/${workspaceId}/runtime`,
      headers: {
        Authorization: `Bearer ${token}`
      },
      payload: {
        runtime_base_url: "http://127.0.0.1:19001"
      }
    });

    expect(response.statusCode).toBe(200);
    expect(response.json()).toMatchObject({
      workspace_id: workspaceId,
      runtime_base_url: "http://127.0.0.1:19001",
      runtime_status: expect.stringMatching(/^(online|offline)$/)
    });
  });

  it("returns 403 when role misses workspace:manage", async () => {
    const { token, userId, workspaceId } = await buildAppAndLogin();
    const membership = db!.getMembershipRole(userId, workspaceId);
    expect(membership).toBeTruthy();

    db!.execute(
      "DELETE FROM role_permissions WHERE role_id = ? AND perm_key = 'workspace:manage'",
      membership!.role_id
    );

    const response = await app!.inject({
      method: "POST",
      url: `/v1/admin/workspaces/${workspaceId}/runtime`,
      headers: {
        Authorization: `Bearer ${token}`
      },
      payload: {
        runtime_base_url: "http://127.0.0.1:19001"
      }
    });

    expect(response.statusCode).toBe(403);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_FORBIDDEN"
      }
    });
  });
});
