import { computed } from "vue";
import { defineStore } from "pinia";

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
import { pinia } from "@/shared/stores/pinia";
import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";

type GeneralSettingsStoreState = {
  value: GeneralSettings;
  capability: GeneralSettingsCapability;
  initialized: boolean;
  loading: boolean;
  saving: boolean;
  error: string;
};

const useGeneralSettingsStoreDefinition = defineStore("generalSettings", {
  state: (): GeneralSettingsStoreState => ({
    value: createDefaultGeneralSettings(),
    capability: createDefaultGeneralSettingsCapability(),
    initialized: false,
    loading: false,
    saving: false,
    error: ""
  })
});

export const useGeneralSettingsStateStore = useGeneralSettingsStoreDefinition;
const generalSettingsStore = useGeneralSettingsStoreDefinition(pinia);

export async function initializeGeneralSettings(): Promise<void> {
  if (generalSettingsStore.initialized || generalSettingsStore.loading) {
    return;
  }

  generalSettingsStore.loading = true;
  generalSettingsStore.error = "";

  try {
    const [persisted, capability] = await Promise.all([loadGeneralSettings(), detectGeneralSettingsCapability()]);

    if (persisted) {
      generalSettingsStore.value = persisted;
    } else {
      generalSettingsStore.value = createDefaultGeneralSettings();
    }

    generalSettingsStore.capability = capability;
    await applyGeneralSettingsField("launchOnStartup", generalSettingsStore.value, generalSettingsStore.capability);
    generalSettingsStore.initialized = true;
  } catch (error) {
    generalSettingsStore.value = createDefaultGeneralSettings();
    generalSettingsStore.capability = createDefaultGeneralSettingsCapability();
    generalSettingsStore.initialized = true;
    generalSettingsStore.error = toDisplayError(error);
  } finally {
    generalSettingsStore.loading = false;
  }
}

export function useGeneralSettings() {
  return {
    state: computed(() => generalSettingsStore.value),
    capability: computed(() => generalSettingsStore.capability),
    loading: computed(() => generalSettingsStore.loading),
    saving: computed(() => generalSettingsStore.saving),
    error: computed(() => generalSettingsStore.error),
    updateField: updateGeneralSetting,
    resetAll: resetGeneralSettings
  };
}

export async function updateGeneralSetting<Path extends GeneralSettingsFieldPath>(
  path: Path,
  value: GeneralSettingsFieldValueMap[Path]
): Promise<void> {
  if (!generalSettingsStore.initialized) {
    await initializeGeneralSettings();
  }

  const next = cloneGeneralSettings(generalSettingsStore.value);
  setFieldValue(next, path, value);

  generalSettingsStore.value = next;
  generalSettingsStore.saving = true;
  generalSettingsStore.error = "";

  try {
    await applyGeneralSettingsField(path, generalSettingsStore.value, generalSettingsStore.capability);
    await saveGeneralSettings(generalSettingsStore.value);
    if (path === "defaultProjectDirectory") {
      const workspace = getCurrentWorkspace();
      if (workspace?.mode === "local") {
        await updateCatalogRoot(workspace.id, String(value));
      }
    }
  } catch (error) {
    generalSettingsStore.error = toDisplayError(error);
  } finally {
    generalSettingsStore.saving = false;
  }
}

export async function resetGeneralSettings(): Promise<void> {
  const defaults = createDefaultGeneralSettings();
  generalSettingsStore.value = defaults;
  generalSettingsStore.saving = true;
  generalSettingsStore.error = "";

  try {
    await applyGeneralSettingsField("launchOnStartup", generalSettingsStore.value, generalSettingsStore.capability);
    await saveGeneralSettings(generalSettingsStore.value);
  } catch (error) {
    generalSettingsStore.error = toDisplayError(error);
  } finally {
    generalSettingsStore.saving = false;
  }
}

export function resetGeneralSettingsStoreForTest(): void {
  generalSettingsStore.value = createDefaultGeneralSettings();
  generalSettingsStore.capability = createDefaultGeneralSettingsCapability();
  generalSettingsStore.initialized = false;
  generalSettingsStore.loading = false;
  generalSettingsStore.saving = false;
  generalSettingsStore.error = "";
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
