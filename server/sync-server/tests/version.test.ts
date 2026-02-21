import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { createApp } from "../src/app";
import { SyncDatabase } from "../src/db";

describe("version endpoint", () => {
  let app: Awaited<ReturnType<typeof createApp>> | undefined;

  afterEach(async () => {
    if (app) {
      await app.close();
      app = undefined;
    }
  });

  it("reports runtime version 0.2.0", async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-sync-"));
    const db = new SyncDatabase(path.join(tempDir, "db.sqlite"));
    db.migrate(path.resolve(process.cwd(), "migrations"));
    app = createApp({ db, token: "token" });

    const response = await app.inject({
      method: "GET",
      url: "/v1/version"
    });

    expect(response.statusCode).toBe(200);
    expect(response.json()).toEqual({
      protocol_version: "2.0.0",
      runtime_version: "0.2.0"
    });
  });
});
