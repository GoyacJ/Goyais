import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { HubDatabase } from "../src/db";
import { loadProtocolVersionFromSchema } from "../src/protocol-version";

const migrationsDir = path.resolve(process.cwd(), "migrations");

describe("hub-server schema migrations", () => {
  let app: Awaited<ReturnType<typeof createApp>> | undefined;

  afterEach(async () => {
    if (app) {
      await app.close();
      app = undefined;
    }
  });

  it("applies base + domain migrations and seeds required data", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-hub-"));
    const db = new HubDatabase(path.join(tempDir, "hub.sqlite"));

    db.migrate(migrationsDir);
    db.migrate(migrationsDir);

    const userCount = db.scalar<number>("SELECT COUNT(*) FROM users");
    const permsCount = db.scalar<number>("SELECT COUNT(*) FROM permissions");
    const menusCount = db.scalar<number>("SELECT COUNT(*) FROM menus");
    const projectsCount = db.scalar<number>("SELECT COUNT(*) FROM projects");
    const modelConfigsCount = db.scalar<number>("SELECT COUNT(*) FROM model_configs");
    const secretsCount = db.scalar<number>("SELECT COUNT(*) FROM secrets");
    const runtimesCount = db.scalar<number>("SELECT COUNT(*) FROM workspace_runtimes");
    const runIndexCount = db.scalar<number>("SELECT COUNT(*) FROM run_index");
    const auditIndexCount = db.scalar<number>("SELECT COUNT(*) FROM audit_index");
    const migrationCount = db.scalar<number>("SELECT COUNT(*) FROM schema_migrations");
    const setupCompleted = db.scalar<number>(
      "SELECT setup_completed FROM system_state WHERE singleton_id = 1"
    );

    expect(userCount).toBe(0);
    expect(permsCount).toBeGreaterThanOrEqual(13);
    expect(menusCount).toBe(5);
    expect(projectsCount).toBe(0);
    expect(modelConfigsCount).toBe(0);
    expect(secretsCount).toBe(0);
    expect(runtimesCount).toBe(0);
    expect(runIndexCount).toBe(0);
    expect(auditIndexCount).toBe(0);
    expect(migrationCount).toBeGreaterThanOrEqual(3);
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
      version: "0.2.0"
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
      version: "0.2.0",
      protocol_version: loadProtocolVersionFromSchema()
    });
  });
});
