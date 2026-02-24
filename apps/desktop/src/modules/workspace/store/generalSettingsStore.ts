import { computed, reactive } from "vue";

import { updateCatalogRoot } from "@/modules/resource/services";
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
import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";

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
    if (path === "defaultProjectDirectory") {
      const workspace = getCurrentWorkspace();
      if (workspace?.mode === "local") {
        await updateCatalogRoot(workspace.id, String(value));
      }
    }
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

const generalSettingsFieldUpdaters: Record<GeneralSettingsFieldPath, (target: GeneralSettings, value: unknown) => void> = {
  launchOnStartup: (target, value) => {
    target.launchOnStartup = value as boolean;
  },
  defaultProjectDirectory: (target, value) => {
    target.defaultProjectDirectory = value as string;
  },
  "notifications.reconnect": (target, value) => {
    target.notifications.reconnect = value as boolean;
  },
  "notifications.approval": (target, value) => {
    target.notifications.approval = value as boolean;
  },
  "notifications.error": (target, value) => {
    target.notifications.error = value as boolean;
  },
  telemetryLevel: (target, value) => {
    target.telemetryLevel = value as GeneralSettings["telemetryLevel"];
  },
  "updatePolicy.channel": (target, value) => {
    target.updatePolicy.channel = value as GeneralSettings["updatePolicy"]["channel"];
  },
  "updatePolicy.checkFrequency": (target, value) => {
    target.updatePolicy.checkFrequency = value as GeneralSettings["updatePolicy"]["checkFrequency"];
  },
  "updatePolicy.autoDownload": (target, value) => {
    target.updatePolicy.autoDownload = value as boolean;
  },
  "diagnostics.level": (target, value) => {
    target.diagnostics.level = value as GeneralSettings["diagnostics"]["level"];
  },
  "diagnostics.logRetentionDays": (target, value) => {
    target.diagnostics.logRetentionDays = value as GeneralSettings["diagnostics"]["logRetentionDays"];
  }
};

function setFieldValue<Path extends GeneralSettingsFieldPath>(
  target: GeneralSettings,
  path: Path,
  value: GeneralSettingsFieldValueMap[Path]
): void {
  generalSettingsFieldUpdaters[path](target, value);
}
