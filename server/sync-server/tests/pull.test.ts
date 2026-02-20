import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { SyncDatabase } from "../src/db";

describe("pull", () => {
  it("returns incremental events", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-sync-"));
    const db = new SyncDatabase(path.join(tempDir, "db.sqlite"));
    db.migrate(path.resolve(process.cwd(), "migrations"));

    db.push({
      device_id: "dev-1",
      since_global_seq: 0,
      events: [
        {
          protocol_version: "1.0.0",
          event_id: "evt-1",
          run_id: "run-1",
          seq: 1,
          ts: "2026-02-20T00:00:00Z",
          type: "plan",
          payload: { summary: "first" }
        },
        {
          protocol_version: "1.0.0",
          event_id: "evt-2",
          run_id: "run-1",
          seq: 2,
          ts: "2026-02-20T00:00:01Z",
          type: "done",
          payload: { status: "completed" }
        }
      ],
      artifacts_meta: []
    });

    const pull = db.pull(0);
    expect(pull.events).toHaveLength(2);
    expect(pull.events[0].event_id).toBe("evt-1");
  });
});
