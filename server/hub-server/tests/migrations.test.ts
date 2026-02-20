import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { HubDatabase } from "../src/db";

const migrationsDir = path.resolve(process.cwd(), "migrations");

describe("hub-server bootstrap scaffold", () => {
  let app: Awaited<ReturnType<typeof createApp>> | undefined;

  afterEach(async () => {
    if (app) {
      await app.close();
      app = undefined;
    }
  });

  it("applies 0001 migration and seeds required data", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-hub-"));
    const db = new HubDatabase(path.join(tempDir, "hub.sqlite"));

    db.migrate(migrationsDir);

    const userCount = db.scalar<number>("SELECT COUNT(*) FROM users");
    const permsCount = db.scalar<number>("SELECT COUNT(*) FROM permissions");
    const menusCount = db.scalar<number>("SELECT COUNT(*) FROM menus");
    const setupCompleted = db.scalar<number>(
      "SELECT setup_completed FROM system_state WHERE singleton_id = 1"
    );

    expect(userCount).toBe(0);
    expect(permsCount).toBeGreaterThanOrEqual(11);
    expect(menusCount).toBe(5);
    expect(setupCompleted).toBe(0);
  });

  it("serves health/version and always returns X-Trace-Id", async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-hub-"));
    const db = new HubDatabase(path.join(tempDir, "hub.sqlite"));
    db.migrate(migrationsDir);

    app = createApp({ db });

    const health = await app.inject({ method: "GET", url: "/v1/health" });
    expect(health.statusCode).toBe(200);
    expect(health.headers["x-trace-id"]).toBeTruthy();
    expect(health.json()).toMatchObject({
      ok: true,
      service: "hub-server",
      version: "0.1.0"
    });

    const version = await app.inject({
      method: "GET",
      url: "/v1/version",
      headers: {
        "X-Trace-Id": "trace-hub-1"
      }
    });

    expect(version.statusCode).toBe(200);
    expect(version.headers["x-trace-id"]).toBe("trace-hub-1");
    expect(version.json()).toMatchObject({
      service: "hub-server",
      version: "0.1.0",
      protocol_version: "1.0.0"
    });
  });
});
