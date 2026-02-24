export type TelemetryLevel = "minimized" | "standard" | "off";
export type UpdateChannel = "stable" | "preview";
export type UpdateCheckFrequency = "manual" | "daily" | "weekly";
export type DiagnosticsLevel = "basic" | "verbose";
export type LogRetentionDays = 7 | 14 | 30;

export type GeneralSettings = {
  launchOnStartup: boolean;
  defaultProjectDirectory: string;
  notifications: {
    reconnect: boolean;
    approval: boolean;
    error: boolean;
  };
  telemetryLevel: TelemetryLevel;
  updatePolicy: {
    channel: UpdateChannel;
    checkFrequency: UpdateCheckFrequency;
    autoDownload: boolean;
  };
  diagnostics: {
    level: DiagnosticsLevel;
    logRetentionDays: LogRetentionDays;
  };
};

export type GeneralSettingsFieldPath =
  | "launchOnStartup"
  | "defaultProjectDirectory"
  | "notifications.reconnect"
  | "notifications.approval"
  | "notifications.error"
  | "telemetryLevel"
  | "updatePolicy.channel"
  | "updatePolicy.checkFrequency"
  | "updatePolicy.autoDownload"
  | "diagnostics.level"
  | "diagnostics.logRetentionDays";

export type GeneralSettingsFieldValueMap = {
  launchOnStartup: boolean;
  defaultProjectDirectory: string;
  "notifications.reconnect": boolean;
  "notifications.approval": boolean;
  "notifications.error": boolean;
  telemetryLevel: TelemetryLevel;
  "updatePolicy.channel": UpdateChannel;
  "updatePolicy.checkFrequency": UpdateCheckFrequency;
  "updatePolicy.autoDownload": boolean;
  "diagnostics.level": DiagnosticsLevel;
  "diagnostics.logRetentionDays": LogRetentionDays;
};

export type GeneralSettingsCapabilityEntry = {
  supported: boolean;
  reasonKey: string;
};

export type GeneralSettingsCapability = {
  launchOnStartup: GeneralSettingsCapabilityEntry;
  notifications: GeneralSettingsCapabilityEntry;
};

export const GENERAL_SETTINGS_STORAGE_KEY = "goyais.settings.general.v1";

const DEFAULT_PROJECT_DIRECTORY = "~/.goyais";

export function createDefaultGeneralSettings(): GeneralSettings {
  return {
    launchOnStartup: true,
    defaultProjectDirectory: DEFAULT_PROJECT_DIRECTORY,
    notifications: {
      reconnect: true,
      approval: true,
      error: true
    },
    telemetryLevel: "minimized",
    updatePolicy: {
      channel: "stable",
      checkFrequency: "daily",
      autoDownload: false
    },
    diagnostics: {
      level: "basic",
      logRetentionDays: 14
    }
  };
}

export function createDefaultGeneralSettingsCapability(): GeneralSettingsCapability {
  return {
    launchOnStartup: {
      supported: false,
      reasonKey: "settings.general.unsupported.platform"
    },
    notifications: {
      supported: false,
      reasonKey: "settings.general.unsupported.notifications"
    }
  };
}

export function cloneGeneralSettings(value: GeneralSettings): GeneralSettings {
  return {
    launchOnStartup: value.launchOnStartup,
    defaultProjectDirectory: value.defaultProjectDirectory,
    notifications: {
      reconnect: value.notifications.reconnect,
      approval: value.notifications.approval,
      error: value.notifications.error
    },
    telemetryLevel: value.telemetryLevel,
    updatePolicy: {
      channel: value.updatePolicy.channel,
      checkFrequency: value.updatePolicy.checkFrequency,
      autoDownload: value.updatePolicy.autoDownload
    },
    diagnostics: {
      level: value.diagnostics.level,
      logRetentionDays: value.diagnostics.logRetentionDays
    }
  };
}

export function normalizeGeneralSettings(raw: unknown): GeneralSettings {
  const defaults = createDefaultGeneralSettings();
  if (!isRecord(raw)) {
    return defaults;
  }

  const notifications = isRecord(raw.notifications) ? raw.notifications : {};
  const updatePolicy = isRecord(raw.updatePolicy) ? raw.updatePolicy : {};
  const diagnostics = isRecord(raw.diagnostics) ? raw.diagnostics : {};

  return {
    launchOnStartup: asBoolean(raw.launchOnStartup, defaults.launchOnStartup),
    defaultProjectDirectory: asString(raw.defaultProjectDirectory, defaults.defaultProjectDirectory),
    notifications: {
      reconnect: asBoolean(notifications.reconnect, defaults.notifications.reconnect),
      approval: asBoolean(notifications.approval, defaults.notifications.approval),
      error: asBoolean(notifications.error, defaults.notifications.error)
    },
    telemetryLevel: asTelemetryLevel(raw.telemetryLevel, defaults.telemetryLevel),
    updatePolicy: {
      channel: asUpdateChannel(updatePolicy.channel, defaults.updatePolicy.channel),
      checkFrequency: asUpdateCheckFrequency(updatePolicy.checkFrequency, defaults.updatePolicy.checkFrequency),
      autoDownload: asBoolean(updatePolicy.autoDownload, defaults.updatePolicy.autoDownload)
    },
    diagnostics: {
      level: asDiagnosticsLevel(diagnostics.level, defaults.diagnostics.level),
      logRetentionDays: asLogRetentionDays(diagnostics.logRetentionDays, defaults.diagnostics.logRetentionDays)
    }
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function asString(value: unknown, fallback: string): string {
  return typeof value === "string" && value.trim() !== "" ? value : fallback;
}

function asBoolean(value: unknown, fallback: boolean): boolean {
  return typeof value === "boolean" ? value : fallback;
}

function asTelemetryLevel(value: unknown, fallback: TelemetryLevel): TelemetryLevel {
  if (value === "minimized" || value === "standard" || value === "off") {
    return value;
  }
  return fallback;
}

function asUpdateChannel(value: unknown, fallback: UpdateChannel): UpdateChannel {
  if (value === "stable" || value === "preview") {
    return value;
  }
  return fallback;
}

function asUpdateCheckFrequency(value: unknown, fallback: UpdateCheckFrequency): UpdateCheckFrequency {
  if (value === "manual" || value === "daily" || value === "weekly") {
    return value;
  }
  return fallback;
}

function asDiagnosticsLevel(value: unknown, fallback: DiagnosticsLevel): DiagnosticsLevel {
  if (value === "basic" || value === "verbose") {
    return value;
  }
  return fallback;
}

function asLogRetentionDays(value: unknown, fallback: LogRetentionDays): LogRetentionDays {
  if (value === 7 || value === 14 || value === 30) {
    return value;
  }
  return fallback;
}
