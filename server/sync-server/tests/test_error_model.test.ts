import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { SyncDatabase } from "../src/db";

describe("error model", () => {
  let app: Awaited<ReturnType<typeof createApp>> | undefined;

  afterEach(async () => {
    if (app) {
      await app.close();
      app = undefined;
    }
  });

  it("returns GoyaisError when auth fails", async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-sync-"));
    const db = new SyncDatabase(path.join(tempDir, "db.sqlite"));
    db.migrate(path.resolve(process.cwd(), "migrations"));
    app = createApp({ db, token: "token" });

    const response = await app.inject({
      method: "GET",
      url: "/v1/sync/pull?since_server_seq=0"
    });

    expect(response.statusCode).toBe(401);
    const body = response.json() as Record<string, unknown>;
    expect(body.error).toBeTypeOf("object");
    const error = body.error as Record<string, unknown>;
    expect(error.code).toBe("E_SYNC_AUTH");
    expect(error.message).toBeTypeOf("string");
    expect(error.trace_id).toBeTypeOf("string");
    expect(error.retryable).toBe(false);
  });
});
