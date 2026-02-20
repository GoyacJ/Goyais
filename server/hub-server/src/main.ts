import path from "node:path";

import { createApp } from "./app";
import { loadConfig } from "./config";
import { HubDatabase } from "./db";
import { loadProtocolVersionFromSchema } from "./protocol-version";

const config = loadConfig();

const db = new HubDatabase(config.dbPath);
const migrationsDir = path.resolve(process.cwd(), "migrations");
db.migrate(migrationsDir);

if (process.argv.includes("--migrate-only")) {
  process.exit(0);
}

const app = createApp({
  db,
  bootstrapToken: config.bootstrapToken,
  hubSecretKey: config.hubSecretKey,
  hubRuntimeSharedSecret: config.hubRuntimeSharedSecret,
  allowPublicSignup: config.allowPublicSignup,
  tokenTtlSeconds: config.tokenTtlSeconds,
  protocolVersion: loadProtocolVersionFromSchema()
});

app.listen({ host: config.host, port: config.port }).catch((error) => {
  app.log.error(error);
  process.exit(1);
});
