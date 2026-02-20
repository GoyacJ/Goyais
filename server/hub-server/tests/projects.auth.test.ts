import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import {
  requireDomainAuth,
  requirePermission,
  requireWorkspaceIdQuery,
  requireWorkspaceMember
} from "../src/auth/workspace-rbac";
import { HubDatabase } from "../src/db";

const migrationsDir = path.resolve(process.cwd(), "migrations");

describe("projects auth/rbac baseline", () => {
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

    app.get("/v1/_test/projects-auth", async (request) => {
      const user = requireDomainAuth(request, db!);
      const workspaceId = requireWorkspaceIdQuery(request);
      const membership = requireWorkspaceMember(request, db!, user, workspaceId);
      requirePermission(db!, membership.role_id, "project:read");
      return {
        ok: true,
        workspace_id: workspaceId
      };
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

  it("returns E_UNAUTHORIZED when token is missing", async () => {
    const { workspaceId } = await buildAppAndLogin();

    const response = await app!.inject({
      method: "GET",
      url: `/v1/_test/projects-auth?workspace_id=${workspaceId}`
    });

    expect(response.statusCode).toBe(401);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_UNAUTHORIZED"
      }
    });
  });

  it("returns E_VALIDATION when workspace_id query is missing", async () => {
    const { token } = await buildAppAndLogin();

    const response = await app!.inject({
      method: "GET",
      url: "/v1/_test/projects-auth",
      headers: {
        Authorization: `Bearer ${token}`
      }
    });

    expect(response.statusCode).toBe(400);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_VALIDATION"
      }
    });
  });

  it("returns E_FORBIDDEN when requester is not workspace member", async () => {
    const { token } = await buildAppAndLogin();

    const response = await app!.inject({
      method: "GET",
      url: "/v1/_test/projects-auth?workspace_id=ws-not-member",
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

  it("returns E_FORBIDDEN when role misses required project permission", async () => {
    const { token, workspaceId } = await buildAppAndLogin();

    const me = await app!.inject({
      method: "GET",
      url: "/v1/me",
      headers: {
        Authorization: `Bearer ${token}`
      }
    });
    const userId = (me.json() as { user: { user_id: string } }).user.user_id;
    const ownerMembership = db!.getMembershipRole(userId, workspaceId);
    expect(ownerMembership).toBeTruthy();

    db!.execute(
      "DELETE FROM role_permissions WHERE role_id = ? AND perm_key = 'project:read'",
      ownerMembership!.role_id
    );

    const response = await app!.inject({
      method: "GET",
      url: `/v1/_test/projects-auth?workspace_id=${workspaceId}`,
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

  it("returns 200 when user has workspace membership and project:read", async () => {
    const { token, workspaceId } = await buildAppAndLogin();

    const response = await app!.inject({
      method: "GET",
      url: `/v1/_test/projects-auth?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      }
    });

    expect(response.statusCode).toBe(200);
    expect(response.json()).toMatchObject({
      ok: true,
      workspace_id: workspaceId
    });
  });
});
