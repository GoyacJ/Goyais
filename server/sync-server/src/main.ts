import path from "node:path";

import { createApp } from "./app";
import { SyncDatabase } from "./db";

const port = Number(process.env.SYNC_SERVER_PORT ?? 8140);
const host = process.env.SYNC_SERVER_HOST ?? "127.0.0.1";
const token = process.env.SYNC_SERVER_TOKEN ?? "change-me";
const dbPath = process.env.SYNC_SERVER_DB_PATH ?? ".goyais/sync-server.db";

const db = new SyncDatabase(dbPath);
const migrationsDir = path.resolve(process.cwd(), "migrations");
db.migrate(migrationsDir);

if (process.argv.includes("--migrate-only")) {
  process.exit(0);
}

const app = createApp({ db, token });

app.listen({ host, port }).catch((error) => {
  app.log.error(error);
  process.exit(1);
});
