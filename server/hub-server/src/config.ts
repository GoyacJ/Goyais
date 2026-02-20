export interface HubServerConfig {
  dbPath: string;
  bootstrapToken: string;
  hubSecretKey: string;
  allowPublicSignup: boolean;
  tokenTtlSeconds: number;
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
  const tokenTtlSeconds = Number(env.GOYAIS_TOKEN_TTL_SECONDS ?? 7 * 24 * 60 * 60);

  return {
    dbPath: env.GOYAIS_HUB_DB_PATH ?? "./data/hub.sqlite",
    bootstrapToken: env.GOYAIS_BOOTSTRAP_TOKEN ?? "",
    hubSecretKey: env.GOYAIS_HUB_SECRET_KEY ?? "",
    allowPublicSignup: parseBoolean(env.GOYAIS_ALLOW_PUBLIC_SIGNUP, false),
    tokenTtlSeconds: Number.isFinite(tokenTtlSeconds) && tokenTtlSeconds > 0 ? tokenTtlSeconds : 7 * 24 * 60 * 60,
    port: Number(env.GOYAIS_SERVER_PORT ?? 8787),
    host: env.GOYAIS_SERVER_HOST ?? "127.0.0.1"
  };
}
