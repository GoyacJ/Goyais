const rawVersion = import.meta.env.VITE_APP_VERSION;
const normalizedVersion = typeof rawVersion === "string" ? rawVersion.trim().replace(/^v/i, "") : "";

export const APP_VERSION = normalizedVersion === "" ? "0.0.0-dev" : normalizedVersion;
