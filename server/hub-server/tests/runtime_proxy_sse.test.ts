import fs from "node:fs";
import { createServer } from "node:http";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { HubDatabase } from "../src/db";

const migrationsDir = path.resolve(process.cwd(), "migrations");

describe("runtime proxy sse passthrough", () => {
  let app: Awaited<ReturnType<typeof createApp>> | undefined;
  let db: HubDatabase | undefined;
  let runtimeServer: ReturnType<typeof createServer> | undefined;
  let runtimeBaseUrl: string | undefined;
  let lastSeenHubAuth: string | undefined;
  let lastSeenUserId: string | undefined;
  let lastSeenTraceId: string | undefined;

  afterEach(async () => {
    if (runtimeServer) {
      await new Promise<void>((resolve, reject) => {
        runtimeServer!.close((error) => {
          if (error) {
            reject(error);
            return;
          }
          resolve();
        });
      });
      runtimeServer = undefined;
      runtimeBaseUrl = undefined;
    }
    if (app) {
      await app.close();
      app = undefined;
    }
  });

  async function startRuntimeServer(workspaceId: string) {
    runtimeServer = createServer((req, res) => {
      if (req.url?.startsWith("/v1/health")) {
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ ok: true, workspace_id: workspaceId, protocol_version: "2.0.0" }));
        return;
      }

      if (req.url?.startsWith("/v1/runs/run-1/events")) {
        lastSeenHubAuth = req.headers["x-hub-auth"] as string | undefined;
        lastSeenUserId = req.headers["x-user-id"] as string | undefined;
        lastSeenTraceId = req.headers["x-trace-id"] as string | undefined;
        res.writeHead(200, {
          "Content-Type": "text/event-stream",
          "Cache-Control": "no-cache",
          Connection: "keep-alive"
        });
        res.write("data: {\"event_id\":\"evt-1\",\"type\":\"plan\"}\n\n");
        setTimeout(() => {
          res.write("data: {\"event_id\":\"evt-2\",\"type\":\"done\"}\n\n");
          res.end();
        }, 20);
        return;
      }

      res.writeHead(404, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: "not_found" }));
    });

    await new Promise<void>((resolve) => {
      runtimeServer!.listen(0, "127.0.0.1", () => resolve());
    });

    const address = runtimeServer.address();
    if (!address || typeof address === "string") {
      throw new Error("Failed to start runtime server");
    }

    runtimeBaseUrl = `http://127.0.0.1:${address.port}`;
  }

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

  it("streams upstream SSE chunks without buffering and injects hub headers", async () => {
    const { token, userId, workspaceId } = await buildAppAndLogin();
    await startRuntimeServer(workspaceId);
    db!.upsertWorkspaceRuntime({
      workspaceId,
      runtimeBaseUrl: runtimeBaseUrl!,
      runtimeStatus: "online"
    });

    const response = await app!.inject({
      method: "GET",
      url: `/v1/runtime/runs/run-1/events?workspace_id=${workspaceId}`,
      headers: {
        Authorization: `Bearer ${token}`
      }
    });

    expect(response.statusCode).toBe(200);
    expect(response.headers["content-type"]).toContain("text/event-stream");
    expect(response.headers["x-trace-id"]).toBeTruthy();
    expect(response.body).toContain("\"event_id\":\"evt-1\"");
    expect(response.body).toContain("\"event_id\":\"evt-2\"");

    expect(lastSeenHubAuth).toBe("hub-runtime-secret");
    expect(lastSeenUserId).toBe(userId);
    expect(lastSeenTraceId).toBeTruthy();
  });
});
