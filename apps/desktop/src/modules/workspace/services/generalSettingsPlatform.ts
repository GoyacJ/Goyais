import {
  createDefaultGeneralSettingsCapability,
  type GeneralSettings,
  type GeneralSettingsCapability,
  type GeneralSettingsFieldPath,
  type UpdateCheckFrequency
} from "@/modules/workspace/schemas/generalSettings";
import desktopPackageJson from "../../../../package.json";

type AutostartAdapter = {
  enable: () => Promise<void>;
  disable: () => Promise<void>;
  isEnabled?: () => Promise<boolean>;
};

type AutostartModule = {
  enable: () => Promise<void>;
  disable: () => Promise<void>;
  isEnabled?: () => Promise<boolean>;
};

type AppModule = {
  getVersion: () => Promise<string>;
};

type UpdaterRelease = {
  version?: string;
};

type UpdaterModule = {
  check: () => Promise<UpdaterRelease | null>;
};

export type AppVersionCheckResult =
  | { status: "latest" }
  | { status: "update-available"; version: string }
  | { status: "unsupported" }
  | { status: "failed"; errorMessage: string };

const dynamicImportModule = new Function("specifier", "return import(specifier);") as (
  specifier: string
) => Promise<unknown>;

const FALLBACK_APP_VERSION =
  typeof desktopPackageJson.version === "string" && desktopPackageJson.version.trim() !== ""
    ? desktopPackageJson.version
    : "0.4.0";

let cachedAutostartPromise: Promise<AutostartAdapter | null> | null = null;
let cachedAppVersionPromise: Promise<string> | null = null;
let cachedUpdaterModulePromise: Promise<UpdaterModule | null> | null = null;

export async function detectGeneralSettingsCapability(): Promise<GeneralSettingsCapability> {
  const capability = createDefaultGeneralSettingsCapability();

  const autostart = await getAutostartAdapter();
  if (autostart) {
    capability.launchOnStartup.supported = true;
    capability.launchOnStartup.reasonKey = "";
  }

  if (canUseNotificationApi()) {
    capability.notifications.supported = true;
    capability.notifications.reasonKey = "";
  }

  return capability;
}

export async function applyGeneralSettingsField(
  path: GeneralSettingsFieldPath,
  state: GeneralSettings,
  capability: GeneralSettingsCapability
): Promise<void> {
  if (path !== "launchOnStartup" || !capability.launchOnStartup.supported) {
    return;
  }

  const autostart = await getAutostartAdapter();
  if (!autostart) {
    return;
  }

  if (state.launchOnStartup) {
    await autostart.enable();
    return;
  }

  await autostart.disable();
}

export async function getCurrentAppVersion(): Promise<string> {
  if (cachedAppVersionPromise) {
    return cachedAppVersionPromise;
  }

  cachedAppVersionPromise = loadCurrentAppVersion();
  return cachedAppVersionPromise;
}

export async function canCheckForAppUpdate(): Promise<boolean> {
  const updater = await getUpdaterModule();
  return updater !== null;
}

export async function checkForAppUpdate(): Promise<AppVersionCheckResult> {
  const updater = await getUpdaterModule();
  if (!updater) {
    return {
      status: "unsupported"
    };
  }

  try {
    const release = await updater.check();
    if (release === null) {
      return {
        status: "latest"
      };
    }

    return {
      status: "update-available",
      version: normalizeVersionValue(release.version)
    };
  } catch (error) {
    return {
      status: "failed",
      errorMessage: error instanceof Error ? error.message : "unknown error"
    };
  }
}

export function resolveUpdatePolicyNextCheck(
  checkFrequency: UpdateCheckFrequency,
  now: Date = new Date()
): Date | null {
  if (checkFrequency === "manual") {
    return null;
  }

  const next = new Date(now);
  if (checkFrequency === "daily") {
    next.setDate(next.getDate() + 1);
    return next;
  }

  next.setDate(next.getDate() + 7);
  return next;
}

export function resolveDiagnosticsRetentionDeadline(
  retentionDays: number,
  now: Date = new Date()
): Date {
  const deadline = new Date(now);
  deadline.setDate(deadline.getDate() - retentionDays);
  return deadline;
}

async function getAutostartAdapter(): Promise<AutostartAdapter | null> {
  if (cachedAutostartPromise) {
    return cachedAutostartPromise;
  }

  cachedAutostartPromise = loadAutostartAdapter();
  return cachedAutostartPromise;
}

async function loadAutostartAdapter(): Promise<AutostartAdapter | null> {
  const module = await importAutostartModule();
  if (!module || typeof module.enable !== "function" || typeof module.disable !== "function") {
    return null;
  }

  return {
    enable: module.enable,
    disable: module.disable,
    isEnabled: module.isEnabled
  };
}

async function loadCurrentAppVersion(): Promise<string> {
  const appModule = await importAppModule();
  if (!appModule || typeof appModule.getVersion !== "function") {
    return FALLBACK_APP_VERSION;
  }

  try {
    return await appModule.getVersion();
  } catch {
    return FALLBACK_APP_VERSION;
  }
}

async function getUpdaterModule(): Promise<UpdaterModule | null> {
  if (cachedUpdaterModulePromise) {
    return cachedUpdaterModulePromise;
  }

  cachedUpdaterModulePromise = importUpdaterModule();
  return cachedUpdaterModulePromise;
}

async function importAutostartModule(): Promise<AutostartModule | null> {
  try {
    const module = await dynamicImportModule("@tauri-apps/plugin-autostart");
    return module as AutostartModule;
  } catch {
    return null;
  }
}

async function importAppModule(): Promise<AppModule | null> {
  try {
    const module = await dynamicImportModule("@tauri-apps/api/app");
    return module as AppModule;
  } catch {
    return null;
  }
}

async function importUpdaterModule(): Promise<UpdaterModule | null> {
  try {
    const module = await dynamicImportModule("@tauri-apps/plugin-updater");
    return module as UpdaterModule;
  } catch {
    return null;
  }
}

function normalizeVersionValue(rawVersion: string | undefined): string {
  if (typeof rawVersion !== "string") {
    return FALLBACK_APP_VERSION;
  }

  const trimmedVersion = rawVersion.trim();
  if (trimmedVersion === "") {
    return FALLBACK_APP_VERSION;
  }

  const normalized = trimmedVersion.replace(/^v/i, "");
  return normalized === "" ? FALLBACK_APP_VERSION : normalized;
}

function canUseNotificationApi(): boolean {
  return typeof window !== "undefined" && typeof window.Notification !== "undefined";
}
