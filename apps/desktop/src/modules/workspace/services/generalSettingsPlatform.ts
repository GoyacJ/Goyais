import {
  createDefaultGeneralSettingsCapability,
  type GeneralSettings,
  type GeneralSettingsCapability,
  type GeneralSettingsFieldPath,
  type UpdateCheckFrequency
} from "@/modules/workspace/schemas/generalSettings";

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

let cachedAutostartPromise: Promise<AutostartAdapter | null> | null = null;

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

async function importAutostartModule(): Promise<AutostartModule | null> {
  try {
    const dynamicImport = new Function("specifier", "return import(specifier);") as (specifier: string) => Promise<unknown>;
    const module = await dynamicImport("@tauri-apps/plugin-autostart");
    return module as AutostartModule;
  } catch {
    return null;
  }
}

function canUseNotificationApi(): boolean {
  return typeof window !== "undefined" && typeof window.Notification !== "undefined";
}
