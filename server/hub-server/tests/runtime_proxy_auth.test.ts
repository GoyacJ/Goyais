import fs from "node:fs";
import { createServer } from "node:http";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { HubDatabase } from "../src/db";

const migrationsDir = path.resolve(process.cwd(), "migrations");

async function startMockRuntimeServer(workspaceId: string) {
  const server = createServer((req, res) => {
    if (req.url?.startsWith("/v1/health")) {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(
        JSON.stringify({
          ok: true,
          workspace_id: workspaceId,
          protocol_version: "2.0.0"
        })
      );
      return;
    }

    res.writeHead(404, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ error: "not_found" }));
  });

  await new Promise<void>((resolve) => {
    server.listen(0, "127.0.0.1", () => resolve());
  });
  const address = server.address();
  if (!address || typeof address === "string") {
    throw new Error("Failed to bind mock runtime server.");
  }

  return {
    baseUrl: `http://127.0.0.1:${address.port}`,
    async close() {
      await new Promise<void>((resolve, reject) => {
        server.close((error) => {
          if (error) {
            reject(error);
            return;
          }
          resolve();
        });
      });
    }
  };
}

describe("runtime proxy auth and permission enforcement", () => {
  let app: Awaited<ReturnType<typeof createApp>> | undefined;
  let db: HubDatabase | undefined;
  let runtimeServer: Awaited<ReturnType<typeof startMockRuntimeServer>> | undefined;

  afterEach(async () => {
    if (runtimeServer) {
      await runtimeServer.close();
      runtimeServer = undefined;
    }
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
      allowPublicSignup: false,
      tokenTtlSeconds: 604800,
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

    return { token, userId, workspaceId };
  }

  it("returns 401 when missing token", async () => {
    const { workspaceId } = await buildAppAndLogin();
    const response = await app!.inject({
      method: "POST",
      url: `/v1/runtime/runs?workspace_id=${workspaceId}`,
      payload: {
        project_id: "p1",
        session_id: "s1",
        input: "hi",
        model_config_id: "mc1",
        workspace_path: "/tmp/work",
        options: { use_worktree: false }
      }
    });

    expect(response.statusCode).toBe(401);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_UNAUTHORIZED"
      }
    });
  });

  it("returns 403 for non-member workspace", async () => {
    const { token } = await buildAppAndLogin();
    const response = await app!.inject({
      method: "POST",
      url: "/v1/runtime/runs?workspace_id=ws-not-member",
      headers: {
        Authorization: `Bearer ${token}`
      },
      payload: {
        project_id: "p1",
        session_id: "s1",
        input: "hi",
        model_config_id: "mc1",
        workspace_path: "/tmp/work",
        options: { use_worktree: false }
      }
    });

    expect(response.statusCode).toBe(403);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_FORBIDDEN"
      }
    });
  });

  it("enforces run:create only for run creation", async () => {
    const { token, userId, workspaceId } = await buildAppAndLogin();
    const membership = db!.getMembershipRole(userId, workspaceId);
    expect(membership).toBeTruthy();
    db!.execute("DELETE FROM role_permissions WHERE role_id = ? AND perm_key = 'run:create'", membership!.role_id);

    const response = await app!.inject({
      method: "POST",
      url: `/v1/runtime/runs?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      },
      payload: {
        project_id: "p1",
        session_id: "s1",
        input: "hi",
        model_config_id: "mc1",
        workspace_path: "/tmp/work",
        options: { use_worktree: false }
      }
    });

    expect(response.statusCode).toBe(403);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_FORBIDDEN",
        details: {
          perm_key: "run:create"
        }
      }
    });
  });

  it("requires run:read for runs listing (no run:create fallback)", async () => {
    const { token, userId, workspaceId } = await buildAppAndLogin();
    const membership = db!.getMembershipRole(userId, workspaceId);
    expect(membership).toBeTruthy();
    db!.execute("DELETE FROM role_permissions WHERE role_id = ? AND perm_key = 'run:read'", membership!.role_id);

    const response = await app!.inject({
      method: "GET",
      url: `/v1/runtime/runs?workspace_id=${workspaceId}&session_id=session-demo`,
      headers: {
        Authorization: `Bearer ${token}`
      }
    });

    expect(response.statusCode).toBe(403);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_FORBIDDEN",
        details: {
          perm_key: "run:read"
        }
      }
    });
  });

  it("requires run:read for sessions listing", async () => {
    const { token, userId, workspaceId } = await buildAppAndLogin();
    const membership = db!.getMembershipRole(userId, workspaceId);
    expect(membership).toBeTruthy();
    db!.execute("DELETE FROM role_permissions WHERE role_id = ? AND perm_key = 'run:read'", membership!.role_id);

    const response = await app!.inject({
      method: "GET",
      url: `/v1/runtime/sessions?workspace_id=${workspaceId}&project_id=project-demo`,
      headers: {
        Authorization: `Bearer ${token}`
      }
    });

    expect(response.statusCode).toBe(403);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_FORBIDDEN",
        details: {
          perm_key: "run:read"
        }
      }
    });
  });

  it("requires modelconfig:read for runtime model catalog listing", async () => {
    const { token, userId, workspaceId } = await buildAppAndLogin();
    const membership = db!.getMembershipRole(userId, workspaceId);
    expect(membership).toBeTruthy();
    db!.execute("DELETE FROM role_permissions WHERE role_id = ? AND perm_key = 'modelconfig:read'", membership!.role_id);

    const response = await app!.inject({
      method: "GET",
      url: `/v1/runtime/model-configs/model-config-1/models?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      }
    });

    expect(response.statusCode).toBe(403);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_FORBIDDEN",
        details: {
          perm_key: "modelconfig:read"
        }
      }
    });
  });

  it("requires run:create for session create and rename", async () => {
    const { token, userId, workspaceId } = await buildAppAndLogin();
    const membership = db!.getMembershipRole(userId, workspaceId);
    expect(membership).toBeTruthy();
    db!.execute("DELETE FROM role_permissions WHERE role_id = ? AND perm_key = 'run:create'", membership!.role_id);

    const createResponse = await app!.inject({
      method: "POST",
      url: `/v1/runtime/sessions?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      },
      payload: {
        project_id: "project-demo",
        title: "New thread"
      }
    });

    expect(createResponse.statusCode).toBe(403);
    expect(createResponse.json()).toMatchObject({
      error: {
        code: "E_FORBIDDEN",
        details: {
          perm_key: "run:create"
        }
      }
    });

    const renameResponse = await app!.inject({
      method: "PATCH",
      url: `/v1/runtime/sessions/session-1?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      },
      payload: {
        title: "Renamed thread"
      }
    });

    expect(renameResponse.statusCode).toBe(403);
    expect(renameResponse.json()).toMatchObject({
      error: {
        code: "E_FORBIDDEN",
        details: {
          perm_key: "run:create"
        }
      }
    });
  });

  it("requires confirm:write for tool confirmations (no run:create fallback)", async () => {
    const { token, userId, workspaceId } = await buildAppAndLogin();
    const membership = db!.getMembershipRole(userId, workspaceId);
    expect(membership).toBeTruthy();
    db!.execute("DELETE FROM role_permissions WHERE role_id = ? AND perm_key = 'confirm:write'", membership!.role_id);

    const response = await app!.inject({
      method: "POST",
      url: `/v1/runtime/tool-confirmations?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      },
      payload: {
        run_id: "run-1",
        call_id: "call-1",
        approved: true
      }
    });

    expect(response.statusCode).toBe(403);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_FORBIDDEN",
        details: {
          perm_key: "confirm:write"
        }
      }
    });
  });

  it("rejects runtime traffic when health workspace_id mismatches registry", async () => {
    const { token, workspaceId } = await buildAppAndLogin();
    runtimeServer = await startMockRuntimeServer("ws-other");
    db!.upsertWorkspaceRuntime({
      workspaceId,
      runtimeBaseUrl: runtimeServer.baseUrl,
      runtimeStatus: "online"
    });

    const response = await app!.inject({
      method: "GET",
      url: `/v1/runtime/health?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      }
    });

    expect(response.statusCode).toBe(409);
    expect(response.json()).toMatchObject({
      error: {
        code: "E_RUNTIME_MISCONFIGURED"
      }
    });
  });
});
