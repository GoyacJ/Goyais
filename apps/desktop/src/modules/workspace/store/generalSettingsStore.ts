import { computed, reactive } from "vue";

import {
  cloneGeneralSettings,
  createDefaultGeneralSettings,
  createDefaultGeneralSettingsCapability,
  type GeneralSettings,
  type GeneralSettingsCapability,
  type GeneralSettingsFieldPath,
  type GeneralSettingsFieldValueMap
} from "@/modules/workspace/schemas/generalSettings";
import { loadGeneralSettings, saveGeneralSettings } from "@/modules/workspace/services/generalSettingsPersistence";
import { applyGeneralSettingsField, detectGeneralSettingsCapability } from "@/modules/workspace/services/generalSettingsPlatform";
import { toDisplayError } from "@/shared/services/errorMapper";

type GeneralSettingsStoreState = {
  value: GeneralSettings;
  capability: GeneralSettingsCapability;
  initialized: boolean;
  loading: boolean;
  saving: boolean;
  error: string;
};

const state = reactive<GeneralSettingsStoreState>({
  value: createDefaultGeneralSettings(),
  capability: createDefaultGeneralSettingsCapability(),
  initialized: false,
  loading: false,
  saving: false,
  error: ""
});

export async function initializeGeneralSettings(): Promise<void> {
  if (state.initialized || state.loading) {
    return;
  }

  state.loading = true;
  state.error = "";

  try {
    const [persisted, capability] = await Promise.all([loadGeneralSettings(), detectGeneralSettingsCapability()]);

    if (persisted) {
      state.value = persisted;
    } else {
      state.value = createDefaultGeneralSettings();
    }

    state.capability = capability;
    await applyGeneralSettingsField("launchOnStartup", state.value, state.capability);
    state.initialized = true;
  } catch (error) {
    state.value = createDefaultGeneralSettings();
    state.capability = createDefaultGeneralSettingsCapability();
    state.initialized = true;
    state.error = toDisplayError(error);
  } finally {
    state.loading = false;
  }
}

export function useGeneralSettings() {
  return {
    state: computed(() => state.value),
    capability: computed(() => state.capability),
    loading: computed(() => state.loading),
    saving: computed(() => state.saving),
    error: computed(() => state.error),
    updateField: updateGeneralSetting,
    resetAll: resetGeneralSettings
  };
}

export async function updateGeneralSetting<Path extends GeneralSettingsFieldPath>(
  path: Path,
  value: GeneralSettingsFieldValueMap[Path]
): Promise<void> {
  if (!state.initialized) {
    await initializeGeneralSettings();
  }

  const next = cloneGeneralSettings(state.value);
  setFieldValue(next, path, value);

  state.value = next;
  state.saving = true;
  state.error = "";

  try {
    await applyGeneralSettingsField(path, state.value, state.capability);
    await saveGeneralSettings(state.value);
  } catch (error) {
    state.error = toDisplayError(error);
  } finally {
    state.saving = false;
  }
}

export async function resetGeneralSettings(): Promise<void> {
  const defaults = createDefaultGeneralSettings();
  state.value = defaults;
  state.saving = true;
  state.error = "";

  try {
    await applyGeneralSettingsField("launchOnStartup", state.value, state.capability);
    await saveGeneralSettings(state.value);
  } catch (error) {
    state.error = toDisplayError(error);
  } finally {
    state.saving = false;
  }
}

export function resetGeneralSettingsStoreForTest(): void {
  state.value = createDefaultGeneralSettings();
  state.capability = createDefaultGeneralSettingsCapability();
  state.initialized = false;
  state.loading = false;
  state.saving = false;
  state.error = "";
}

function setFieldValue<Path extends GeneralSettingsFieldPath>(
  target: GeneralSettings,
  path: Path,
  value: GeneralSettingsFieldValueMap[Path]
): void {
  switch (path) {
    case "launchOnStartup":
      target.launchOnStartup = value as boolean;
      return;
    case "defaultProjectDirectory":
      target.defaultProjectDirectory = value as string;
      return;
    case "notifications.reconnect":
      target.notifications.reconnect = value as boolean;
      return;
    case "notifications.approval":
      target.notifications.approval = value as boolean;
      return;
    case "notifications.error":
      target.notifications.error = value as boolean;
      return;
    case "telemetryLevel":
      target.telemetryLevel = value as GeneralSettings["telemetryLevel"];
      return;
    case "updatePolicy.channel":
      target.updatePolicy.channel = value as GeneralSettings["updatePolicy"]["channel"];
      return;
    case "updatePolicy.checkFrequency":
      target.updatePolicy.checkFrequency = value as GeneralSettings["updatePolicy"]["checkFrequency"];
      return;
    case "updatePolicy.autoDownload":
      target.updatePolicy.autoDownload = value as boolean;
      return;
    case "diagnostics.level":
      target.diagnostics.level = value as GeneralSettings["diagnostics"]["level"];
      return;
    case "diagnostics.logRetentionDays":
      target.diagnostics.logRetentionDays = value as GeneralSettings["diagnostics"]["logRetentionDays"];
      return;
  }
}
