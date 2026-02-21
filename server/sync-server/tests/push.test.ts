import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { SyncDatabase } from "../src/db";

describe("push", () => {
  it("inserts events and returns max seq", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-sync-"));
    const db = new SyncDatabase(path.join(tempDir, "db.sqlite"));
    db.migrate(path.resolve(process.cwd(), "migrations"));

    const result = db.push({
      device_id: "dev-1",
      since_global_seq: 0,
      events: [
        {
          protocol_version: "2.0.0",
          trace_id: "trace-1",
          event_id: "evt-1",
          run_id: "run-1",
          seq: 1,
          ts: "2026-02-20T00:00:00Z",
          type: "plan",
          payload: { summary: "s" }
        }
      ],
      artifacts_meta: []
    });

    expect(result.inserted).toBe(1);
    expect(result.max_server_seq).toBeGreaterThan(0);
  });

  it("is idempotent on duplicate event_id", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goyais-sync-"));
    const db = new SyncDatabase(path.join(tempDir, "db.sqlite"));
    db.migrate(path.resolve(process.cwd(), "migrations"));

    const payload = {
      device_id: "dev-1",
      since_global_seq: 0,
      events: [
        {
          protocol_version: "2.0.0" as const,
          trace_id: "trace-dup",
          event_id: "evt-dup-1",
          run_id: "run-1",
          seq: 1,
          ts: "2026-02-20T00:00:00Z",
          type: "plan" as const,
          payload: { summary: "dup" }
        }
      ],
      artifacts_meta: []
    };

    const first = db.push(payload);
    const second = db.push(payload);
    expect(first.inserted).toBe(1);
    expect(second.inserted).toBe(0);
    expect(second.max_server_seq).toBe(first.max_server_seq);
  });
});
