import fs from "node:fs";
import path from "node:path";
import { DatabaseSync } from "node:sqlite";

import type { PushRequest, SyncEventEnvelope } from "./types";

export class SyncDatabase {
  private readonly db: DatabaseSync;

  constructor(filePath: string) {
    fs.mkdirSync(path.dirname(filePath), { recursive: true });
    this.db = new DatabaseSync(filePath);
    this.db.exec("PRAGMA journal_mode = WAL;");
  }

  migrate(migrationsDir: string): void {
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS schema_migrations (
        version TEXT PRIMARY KEY,
        applied_at TEXT NOT NULL
      );
    `);

    const files = fs
      .readdirSync(migrationsDir)
      .filter((file) => file.endsWith(".sql"))
      .sort();

    for (const file of files) {
      const exists = this.db
        .prepare("SELECT 1 FROM schema_migrations WHERE version = ?")
        .get(file);
      if (exists) continue;

      const sql = fs.readFileSync(path.join(migrationsDir, file), "utf8");
      this.db.exec(sql);
      this.db
        .prepare(
          "INSERT INTO schema_migrations(version, applied_at) VALUES(?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))"
        )
        .run(file);
    }
  }

  push(payload: PushRequest): { inserted: number; max_server_seq: number } {
    const insert = this.db.prepare(`
      INSERT OR IGNORE INTO events(
        protocol_version, event_id, run_id, run_seq, ts, type, payload_json, source_device, created_at
      ) VALUES(?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))
    `);

    let inserted = 0;
    for (const event of payload.events) {
      const result = insert.run(
        event.protocol_version,
        event.event_id,
        event.run_id,
        event.seq,
        event.ts,
        event.type,
        JSON.stringify(event.payload),
        payload.device_id
      );
      inserted += Number(result.changes);
    }

    const row = this.db
      .prepare("SELECT COALESCE(MAX(server_seq), 0) AS max_server_seq FROM events")
      .get() as { max_server_seq: number };

    return { inserted, max_server_seq: row.max_server_seq };
  }

  pull(sinceSeq: number): { events: Array<SyncEventEnvelope & { server_seq: number }>; max_server_seq: number } {
    const rows = this.db
      .prepare(
        "SELECT server_seq, protocol_version, event_id, run_id, run_seq, ts, type, payload_json FROM events WHERE server_seq > ? ORDER BY server_seq ASC"
      )
      .all(sinceSeq) as Array<{
      server_seq: number;
      protocol_version: "1.0.0";
      event_id: string;
      run_id: string;
      run_seq: number;
      ts: string;
      type: SyncEventEnvelope["type"];
      payload_json: string;
    }>;

    const events = rows.map((row) => ({
      server_seq: row.server_seq,
      protocol_version: row.protocol_version,
      event_id: row.event_id,
      run_id: row.run_id,
      seq: row.run_seq,
      ts: row.ts,
      type: row.type,
      payload: JSON.parse(row.payload_json) as Record<string, unknown>
    }));

    const max_server_seq = rows.length > 0 ? rows[rows.length - 1].server_seq : sinceSeq;
    return { events, max_server_seq };
  }
}
