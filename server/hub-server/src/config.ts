export interface HubServerConfig {
  dbPath: string;
  bootstrapToken: string;
  allowPublicSignup: boolean;
  port: number;
  host: string;
}

function parseBoolean(value: string | undefined, fallback: boolean): boolean {
  if (value === undefined) {
    return fallback;
  }

  if (value === "true") {
    return true;
  }

  if (value === "false") {
    return false;
  }

  return fallback;
}

export function loadConfig(env: NodeJS.ProcessEnv = process.env): HubServerConfig {
  return {
    dbPath: env.GOYAIS_HUB_DB_PATH ?? "./data/hub.sqlite",
    bootstrapToken: env.GOYAIS_BOOTSTRAP_TOKEN ?? "",
    allowPublicSignup: parseBoolean(env.GOYAIS_ALLOW_PUBLIC_SIGNUP, false),
    port: Number(env.GOYAIS_SERVER_PORT ?? 8787),
    host: env.GOYAIS_SERVER_HOST ?? "127.0.0.1"
  };
}
