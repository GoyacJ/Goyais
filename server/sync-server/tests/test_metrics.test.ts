import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { SyncDatabase } from "../src/db";

describe("metrics endpoint", () => {
  let app: Awaited<ReturnType<typeof createApp>> | undefined;

  afterEach(async () => {
    if (app) {
      await app.close();
      app = undefined;
    }
  });

  it("returns required metrics fields", async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-sync-"));
    const db = new SyncDatabase(path.join(tempDir, "db.sqlite"));
    db.migrate(path.resolve(process.cwd(), "migrations"));
    app = createApp({ db, token: "token" });

    const response = await app.inject({
      method: "GET",
      url: "/v1/metrics"
    });

    expect(response.statusCode).toBe(200);
    const body = response.json() as Record<string, unknown>;
    expect(body.events_total).toBeTypeOf("number");
    expect(body.push_requests_total).toBeTypeOf("number");
    expect(body.pull_requests_total).toBeTypeOf("number");
    expect(body.auth_fail_total).toBeTypeOf("number");
  });
});
