import { computed, onMounted, ref } from "vue";

import {
  canCheckForAppUpdate,
  checkForAppUpdate,
  getCurrentAppVersion
} from "@/modules/workspace/services/generalSettingsPlatform";
import { initializeGeneralSettings, useGeneralSettings } from "@/modules/workspace/store/generalSettingsStore";
import { useI18n } from "@/shared/i18n";

type VersionCheckState = "idle" | "checking" | "latest" | "available" | "unsupported" | "failed";

export function useSettingsGeneralViewModel() {
  const { t } = useI18n();
  const settings = useGeneralSettings();

  const currentVersion = ref("");
  const versionCheckState = ref<VersionCheckState>("idle");
  const availableVersion = ref("");
  const checkVersionError = ref("");

  onMounted(() => {
    void initializeGeneralSettings();
    void initializeVersionState();
  });

  const enabledDisabledOptions = computed(() => [
    { value: "enabled", label: t("settings.general.option.enabled") },
    { value: "disabled", label: t("settings.general.option.disabled") }
  ]);

  const telemetryOptions = computed(() => [
    { value: "minimized", label: t("settings.general.option.telemetry.minimized") },
    { value: "standard", label: t("settings.general.option.telemetry.standard") },
    { value: "off", label: t("settings.general.option.telemetry.off") }
  ]);

  const updateChannelOptions = computed(() => [
    { value: "stable", label: t("settings.general.option.updateChannel.stable") },
    { value: "preview", label: t("settings.general.option.updateChannel.preview") }
  ]);

  const updateFrequencyOptions = computed(() => [
    { value: "manual", label: t("settings.general.option.updateFrequency.manual") },
    { value: "daily", label: t("settings.general.option.updateFrequency.daily") },
    { value: "weekly", label: t("settings.general.option.updateFrequency.weekly") }
  ]);

  const diagnosticsLevelOptions = computed(() => [
    { value: "basic", label: t("settings.general.option.diagnosticsLevel.basic") },
    { value: "verbose", label: t("settings.general.option.diagnosticsLevel.verbose") }
  ]);

  const logRetentionOptions = computed(() => [
    { value: "7", label: t("settings.general.option.logRetention.7") },
    { value: "14", label: t("settings.general.option.logRetention.14") },
    { value: "30", label: t("settings.general.option.logRetention.30") }
  ]);

  const launchOnStartupModel = computed<string>({
    get: () => toEnabledFlag(settings.state.value.launchOnStartup),
    set: (value) => void settings.updateField("launchOnStartup", value === "enabled")
  });

  const defaultDirectoryModel = computed<string>({
    get: () => settings.state.value.defaultProjectDirectory,
    set: (value) => void settings.updateField("defaultProjectDirectory", value)
  });

  const notificationsReconnectModel = computed<string>({
    get: () => toEnabledFlag(settings.state.value.notifications.reconnect),
    set: (value) => void settings.updateField("notifications.reconnect", value === "enabled")
  });

  const notificationsApprovalModel = computed<string>({
    get: () => toEnabledFlag(settings.state.value.notifications.approval),
    set: (value) => void settings.updateField("notifications.approval", value === "enabled")
  });

  const notificationsErrorModel = computed<string>({
    get: () => toEnabledFlag(settings.state.value.notifications.error),
    set: (value) => void settings.updateField("notifications.error", value === "enabled")
  });

  const telemetryLevelModel = computed<string>({
    get: () => settings.state.value.telemetryLevel,
    set: (value) => void settings.updateField("telemetryLevel", value as "minimized" | "standard" | "off")
  });

  const updateChannelModel = computed<string>({
    get: () => settings.state.value.updatePolicy.channel,
    set: (value) => void settings.updateField("updatePolicy.channel", value as "stable" | "preview")
  });

  const updateFrequencyModel = computed<string>({
    get: () => settings.state.value.updatePolicy.checkFrequency,
    set: (value) => void settings.updateField("updatePolicy.checkFrequency", value as "manual" | "daily" | "weekly")
  });

  const updateAutoDownloadModel = computed<string>({
    get: () => toEnabledFlag(settings.state.value.updatePolicy.autoDownload),
    set: (value) => void settings.updateField("updatePolicy.autoDownload", value === "enabled")
  });

  const diagnosticsLevelModel = computed<string>({
    get: () => settings.state.value.diagnostics.level,
    set: (value) => void settings.updateField("diagnostics.level", value as "basic" | "verbose")
  });

  const logRetentionModel = computed<string>({
    get: () => String(settings.state.value.diagnostics.logRetentionDays),
    set: (value) => void settings.updateField("diagnostics.logRetentionDays", Number(value) as 7 | 14 | 30)
  });

  const launchUnsupportedReason = computed(() =>
    settings.capability.value.launchOnStartup.supported ? "" : t(settings.capability.value.launchOnStartup.reasonKey)
  );

  const notificationsUnsupportedReason = computed(() =>
    settings.capability.value.notifications.supported ? "" : t(settings.capability.value.notifications.reasonKey)
  );

  const currentVersionText = computed(() => {
    const version = currentVersion.value.trim();
    if (version === "") {
      return t("settings.general.field.currentVersion.unknown");
    }
    return toVersionBadge(version);
  });

  const checkVersionUnsupportedReason = computed(() =>
    versionCheckState.value === "unsupported" ? t("settings.general.unsupported.updateCheck") : ""
  );

  const checkVersionButtonDisabled = computed(
    () =>
      settings.loading.value ||
      settings.saving.value ||
      versionCheckState.value === "checking" ||
      versionCheckState.value === "unsupported"
  );

  const checkVersionActionLabel = computed(() =>
    versionCheckState.value === "checking"
      ? t("settings.general.field.checkVersion.checking")
      : t("settings.general.field.checkVersion.action")
  );

  const checkVersionHint = computed(() => {
    switch (versionCheckState.value) {
      case "latest":
        return t("settings.general.versionCheck.latest");
      case "available":
        return `${t("settings.general.versionCheck.availablePrefix")} ${toVersionBadge(availableVersion.value)}`;
      case "failed":
        return checkVersionError.value === ""
          ? t("settings.general.versionCheck.failed")
          : `${t("settings.general.versionCheck.failedPrefix")} ${checkVersionError.value}`;
      default:
        return "";
    }
  });

  function resetAll(): void {
    void settings.resetAll();
  }

  async function checkVersion(): Promise<void> {
    if (versionCheckState.value === "checking" || versionCheckState.value === "unsupported") {
      return;
    }

    versionCheckState.value = "checking";
    availableVersion.value = "";
    checkVersionError.value = "";

    const result = await checkForAppUpdate();
    if (result.status === "latest") {
      versionCheckState.value = "latest";
      return;
    }

    if (result.status === "update-available") {
      versionCheckState.value = "available";
      availableVersion.value = result.version;
      return;
    }

    if (result.status === "unsupported") {
      versionCheckState.value = "unsupported";
      return;
    }

    versionCheckState.value = "failed";
    checkVersionError.value = result.errorMessage;
  }

  async function initializeVersionState(): Promise<void> {
    currentVersion.value = await getCurrentAppVersion();

    const canCheckUpdate = await canCheckForAppUpdate();
    if (!canCheckUpdate) {
      versionCheckState.value = "unsupported";
    }
  }

  return {
    t,
    settings,
    enabledDisabledOptions,
    telemetryOptions,
    updateChannelOptions,
    updateFrequencyOptions,
    diagnosticsLevelOptions,
    logRetentionOptions,
    launchOnStartupModel,
    defaultDirectoryModel,
    notificationsReconnectModel,
    notificationsApprovalModel,
    notificationsErrorModel,
    telemetryLevelModel,
    updateChannelModel,
    updateFrequencyModel,
    updateAutoDownloadModel,
    diagnosticsLevelModel,
    logRetentionModel,
    launchUnsupportedReason,
    notificationsUnsupportedReason,
    currentVersionText,
    checkVersionUnsupportedReason,
    checkVersionButtonDisabled,
    checkVersionActionLabel,
    checkVersionHint,
    checkVersion,
    resetAll
  };
}

function toEnabledFlag(value: boolean): string {
  return value ? "enabled" : "disabled";
}

function toVersionBadge(version: string): string {
  const normalized = version.trim().replace(/^v/i, "");
  return normalized === "" ? "" : `v${normalized}`;
}
