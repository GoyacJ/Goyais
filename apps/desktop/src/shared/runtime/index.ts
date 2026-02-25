export type RuntimeTarget = "desktop" | "mobile" | "web";

export type RuntimeCapabilities = {
  supportsLocalWorkspace: boolean;
  supportsSidecar: boolean;
  supportsWindowControls: boolean;
  supportsDirectoryImport: boolean;
  supportsAutostart: boolean;
};

type HubSecurityPolicyInput = {
  runtimeTarget: RuntimeTarget;
  requireHttpsHub: boolean;
  allowInsecureHub: boolean;
  isDev: boolean;
};

type ControlHubResolutionInput = HubSecurityPolicyInput & {
  capabilities: RuntimeCapabilities;
  hubBaseUrl: string;
};

const DEFAULT_DESKTOP_HUB_BASE_URL = "http://127.0.0.1:8787";

let cachedRuntimeTarget: RuntimeTarget | null = null;
let cachedRuntimeCapabilities: RuntimeCapabilities | null = null;

export function resolveRuntimeTarget(rawTarget: string | undefined | null): RuntimeTarget {
  const normalized = (rawTarget ?? "").trim().toLowerCase();
  if (normalized === "mobile") {
    return "mobile";
  }
  if (normalized === "web") {
    return "web";
  }
  return "desktop";
}

export function resolveRuntimeCapabilities(target: RuntimeTarget): RuntimeCapabilities {
  if (target === "mobile") {
    return {
      supportsLocalWorkspace: false,
      supportsSidecar: false,
      supportsWindowControls: false,
      supportsDirectoryImport: false,
      supportsAutostart: false
    };
  }

  if (target === "web") {
    return {
      supportsLocalWorkspace: false,
      supportsSidecar: false,
      supportsWindowControls: false,
      supportsDirectoryImport: true,
      supportsAutostart: false
    };
  }

  return {
    supportsLocalWorkspace: true,
    supportsSidecar: true,
    supportsWindowControls: true,
    supportsDirectoryImport: true,
    supportsAutostart: true
  };
}

export function getRuntimeTarget(): RuntimeTarget {
  if (cachedRuntimeTarget !== null) {
    return cachedRuntimeTarget;
  }
  cachedRuntimeTarget = resolveRuntimeTarget(import.meta.env.VITE_RUNTIME_TARGET);
  return cachedRuntimeTarget;
}

export function getRuntimeCapabilities(): RuntimeCapabilities {
  if (cachedRuntimeCapabilities !== null) {
    return cachedRuntimeCapabilities;
  }
  cachedRuntimeCapabilities = resolveRuntimeCapabilities(getRuntimeTarget());
  return cachedRuntimeCapabilities;
}

export function isRuntimeCapabilitySupported(capability: keyof RuntimeCapabilities): boolean {
  return getRuntimeCapabilities()[capability];
}

export function enforceHubSecurityPolicy(hubUrl: string, input: HubSecurityPolicyInput): void {
  const normalized = normalizeHubBaseUrl(hubUrl);
  const parsed = new URL(normalized);

  if (!input.requireHttpsHub) {
    return;
  }

  if (parsed.protocol === "https:") {
    return;
  }

  if (input.runtimeTarget === "mobile" && input.allowInsecureHub && input.isDev) {
    return;
  }

  throw new Error("Mobile release requires HTTPS hub URL");
}

export function resolveControlHubBaseUrl(input: ControlHubResolutionInput): string {
  const candidate = input.hubBaseUrl.trim();
  const resolved = candidate !== "" ? candidate : input.capabilities.supportsSidecar ? DEFAULT_DESKTOP_HUB_BASE_URL : "";

  if (resolved === "") {
    throw new Error("VITE_HUB_BASE_URL is required when sidecar is unavailable");
  }

  enforceHubSecurityPolicy(resolved, {
    runtimeTarget: input.runtimeTarget,
    requireHttpsHub: input.requireHttpsHub,
    allowInsecureHub: input.allowInsecureHub,
    isDev: input.isDev
  });

  return resolved.replace(/\/$/, "");
}

export function getControlHubBaseUrl(): string {
  const runtimeTarget = getRuntimeTarget();
  const capabilities = getRuntimeCapabilities();
  const hubBaseUrl = import.meta.env.VITE_HUB_BASE_URL ?? "";
  const requireHttpsHub = resolveRequireHttpsHub(runtimeTarget, import.meta.env.VITE_REQUIRE_HTTPS_HUB);
  const allowInsecureHub = resolveBooleanFlag(import.meta.env.VITE_ALLOW_INSECURE_HUB, false);

  return resolveControlHubBaseUrl({
    runtimeTarget,
    capabilities,
    hubBaseUrl,
    requireHttpsHub,
    allowInsecureHub,
    isDev: import.meta.env.DEV
  });
}

export function validateWorkspaceHubUrl(rawHubUrl: string): string {
  const normalized = normalizeHubBaseUrl(rawHubUrl);
  const runtimeTarget = getRuntimeTarget();
  const requireHttpsHub = resolveRequireHttpsHub(runtimeTarget, import.meta.env.VITE_REQUIRE_HTTPS_HUB);
  const allowInsecureHub = resolveBooleanFlag(import.meta.env.VITE_ALLOW_INSECURE_HUB, false);

  enforceHubSecurityPolicy(normalized, {
    runtimeTarget,
    requireHttpsHub,
    allowInsecureHub,
    isDev: import.meta.env.DEV
  });

  return normalized;
}

export function resetRuntimeCacheForTests(): void {
  cachedRuntimeTarget = null;
  cachedRuntimeCapabilities = null;
}

export function normalizeHubBaseUrl(rawHubUrl: string): string {
  const normalized = rawHubUrl.trim();
  if (normalized === "") {
    throw new Error("Hub URL is required");
  }

  let parsed: URL;
  try {
    parsed = new URL(normalized);
  } catch {
    throw new Error(`Invalid hub URL: ${normalized}`);
  }

  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error("Hub URL must use http or https protocol");
  }

  return normalized.replace(/\/$/, "");
}

function resolveRequireHttpsHub(runtimeTarget: RuntimeTarget, rawValue: string | undefined): boolean {
  const fromEnv = resolveOptionalBoolean(rawValue);
  if (fromEnv !== null) {
    return fromEnv;
  }
  if (runtimeTarget === "mobile") {
    return import.meta.env.PROD;
  }
  return false;
}

function resolveBooleanFlag(rawValue: string | undefined, fallback: boolean): boolean {
  const normalized = resolveOptionalBoolean(rawValue);
  return normalized === null ? fallback : normalized;
}

function resolveOptionalBoolean(rawValue: string | undefined): boolean | null {
  const normalized = (rawValue ?? "").trim().toLowerCase();
  if (normalized === "1" || normalized === "true" || normalized === "yes" || normalized === "on") {
    return true;
  }
  if (normalized === "0" || normalized === "false" || normalized === "no" || normalized === "off") {
    return false;
  }
  return null;
}
