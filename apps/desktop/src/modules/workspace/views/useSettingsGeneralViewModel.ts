import { computed, onMounted } from "vue";

import { initializeGeneralSettings, useGeneralSettings } from "@/modules/workspace/store/generalSettingsStore";
import { useI18n } from "@/shared/i18n";

export function useSettingsGeneralViewModel() {
  const { t } = useI18n();
  const settings = useGeneralSettings();

  onMounted(() => {
    void initializeGeneralSettings();
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

  function resetAll(): void {
    void settings.resetAll();
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
    resetAll
  };
}

function toEnabledFlag(value: boolean): string {
  return value ? "enabled" : "disabled";
}
