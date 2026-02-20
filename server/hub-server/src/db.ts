import fs from "node:fs";
import path from "node:path";
import { DatabaseSync, type SQLInputValue } from "node:sqlite";

export class HubDatabase {
  private readonly db: DatabaseSync;

  constructor(filePath: string) {
    fs.mkdirSync(path.dirname(filePath), { recursive: true });
    this.db = new DatabaseSync(filePath);
    this.db.exec("PRAGMA journal_mode = WAL;");
    this.db.exec("PRAGMA foreign_keys = ON;");
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
      const exists = this.db.prepare("SELECT 1 FROM schema_migrations WHERE version = ?").get(file);
      if (exists) {
        continue;
      }

      const sql = fs.readFileSync(path.join(migrationsDir, file), "utf8");
      this.db.exec(sql);
      this.db
        .prepare(
          "INSERT INTO schema_migrations(version, applied_at) VALUES(?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))"
        )
        .run(file);
    }
  }

  scalar<T extends number | string>(sql: string, ...params: SQLInputValue[]): T {
    const row = this.db.prepare(sql).get(...params) as Record<string, T>;
    return row[Object.keys(row)[0]];
  }
}
