import path from "node:path";

import { createApp } from "./app";
import { loadConfig } from "./config";
import { HubDatabase } from "./db";

const config = loadConfig();

const db = new HubDatabase(config.dbPath);
const migrationsDir = path.resolve(process.cwd(), "migrations");
db.migrate(migrationsDir);

if (process.argv.includes("--migrate-only")) {
  process.exit(0);
}

const app = createApp({ db });

app.listen({ host: config.host, port: config.port }).catch((error) => {
  app.log.error(error);
  process.exit(1);
});
