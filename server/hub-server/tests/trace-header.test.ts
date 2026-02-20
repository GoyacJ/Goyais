import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { HubDatabase } from "../src/db";

const migrationsDir = path.resolve(process.cwd(), "migrations");

describe("trace headers", () => {
  let app: Awaited<ReturnType<typeof createApp>> | undefined;

  afterEach(async () => {
    if (app) {
      await app.close();
      app = undefined;
    }
  });

  it("echoes incoming X-Trace-Id", async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-hub-"));
    const db = new HubDatabase(path.join(tempDir, "hub.sqlite"));
    db.migrate(migrationsDir);

    app = createApp({ db, bootstrapToken: "bootstrap-123", allowPublicSignup: false, tokenTtlSeconds: 604800 });

    const response = await app.inject({
      method: "GET",
      url: "/v1/auth/bootstrap/status",
      headers: {
        "X-Trace-Id": "trace-hub-header"
      }
    });

    expect(response.statusCode).toBe(200);
    expect(response.headers["x-trace-id"]).toBe("trace-hub-header");
  });

  it("generates X-Trace-Id when request header is missing", async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-hub-"));
    const db = new HubDatabase(path.join(tempDir, "hub.sqlite"));
    db.migrate(migrationsDir);

    app = createApp({ db, bootstrapToken: "bootstrap-123", allowPublicSignup: false, tokenTtlSeconds: 604800 });

    const response = await app.inject({
      method: "GET",
      url: "/v1/version"
    });

    expect(response.statusCode).toBe(200);
    expect(response.headers["x-trace-id"]).toBeTruthy();
  });
});
