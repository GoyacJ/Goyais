import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { SyncDatabase } from "../src/db";

describe("trace header", () => {
  let app: Awaited<ReturnType<typeof createApp>> | undefined;

  afterEach(async () => {
    if (app) {
      await app.close();
      app = undefined;
    }
  });

  it("echoes incoming X-Trace-Id", async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-sync-"));
    const db = new SyncDatabase(path.join(tempDir, "db.sqlite"));
    db.migrate(path.resolve(process.cwd(), "migrations"));
    app = createApp({ db, token: "token" });

    const response = await app.inject({
      method: "GET",
      url: "/v1/sync/pull?since_server_seq=0",
      headers: {
        Authorization: "Bearer token",
        "X-Trace-Id": "trace-sync-1"
      }
    });

    expect(response.statusCode).toBe(200);
    expect(response.headers["x-trace-id"]).toBe("trace-sync-1");
  });
});
