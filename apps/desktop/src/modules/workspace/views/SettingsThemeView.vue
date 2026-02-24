<template>
  <SettingsShell
    active-key="settings_theme"
    :title="t('menu.settingsTheme')"
    :subtitle="`${t('settings.scope')} / Theme`"
    runtime-status-mode
    :runtime-conversation-status="workspaceStatus.conversationStatus.value"
    :runtime-connection-status="workspaceStatus.connectionStatus.value"
    :runtime-user-display-name="workspaceStatus.userDisplayName.value"
    :runtime-hub-url="workspaceStatus.hubURL.value"
  >
    <section class="theme-layout">
      <article class="config-panel">
        <header class="panel-header">
          <BaseButton data-testid="theme-reset-button" variant="secondary" @click="theme.resetThemeSettings()">
            {{ t("settings.theme.reset") }}
          </BaseButton>
        </header>

        <section class="config-group">
          <div class="group-header">
            <h4>{{ t("settings.theme.modeTitle") }}</h4>
            <p>{{ t("settings.theme.modeHint") }}</p>
          </div>
          <BaseSelect data-testid="theme-mode-select" v-model="themeModeModel" :options="themeModeOptions" />
        </section>

        <section class="config-group">
          <div class="group-header">
            <h4>{{ t("settings.theme.fontStyleTitle") }}</h4>
            <p>{{ t("settings.theme.fontStyleHint") }}</p>
          </div>
          <BaseSelect data-testid="font-style-select" v-model="fontStyleModel" :options="fontStyleOptions" />
        </section>

        <section class="config-group">
          <div class="group-header">
            <h4>{{ t("settings.theme.fontScaleTitle") }}</h4>
            <p>{{ t("settings.theme.fontScaleHint") }}</p>
          </div>
          <BaseSelect data-testid="font-scale-select" v-model="fontScaleModel" :options="fontScaleOptions" />
        </section>

        <section class="config-group">
          <div class="group-header">
            <h4>{{ t("settings.theme.presetTitle") }}</h4>
            <p>{{ t("settings.theme.presetHint") }}</p>
          </div>
          <BaseSelect data-testid="theme-preset-select" v-model="presetModel" :options="presetOptions" />
        </section>
      </article>
    </section>
  </SettingsShell>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { projectStore } from "@/modules/project/store";
import SettingsShell from "@/shared/shells/SettingsShell.vue";
import { useI18n } from "@/shared/i18n";
import { useWorkspaceStatusSync } from "@/shared/stores/workspaceStatusStore";
import BaseButton from "@/shared/ui/BaseButton.vue";
import BaseSelect from "@/shared/ui/BaseSelect.vue";
import { useTheme, type FontScale, type FontStyle, type ThemeMode, type ThemePreset } from "@/shared/stores/themeStore";

const { t } = useI18n();
const theme = useTheme();
const workspaceStatus = useWorkspaceStatusSync({
  conversationId: computed(() => projectStore.activeConversationId)
});

const themeModeModel = computed<ThemeMode>({
  get: () => theme.mode.value,
  set: (value) => theme.setThemeMode(value)
});

const fontStyleModel = computed<FontStyle>({
  get: () => theme.fontStyle.value,
  set: (value) => theme.setFontStyle(value)
});

const fontScaleModel = computed<FontScale>({
  get: () => theme.fontScale.value,
  set: (value) => theme.setFontScale(value)
});

const presetModel = computed<ThemePreset>({
  get: () => theme.preset.value,
  set: (value) => theme.setThemePreset(value)
});

const themeModeOptions = computed(() => [
  { value: "system", label: t("settings.theme.system") },
  { value: "dark", label: t("settings.theme.dark") },
  { value: "light", label: t("settings.theme.light") }
]);

const fontStyleOptions = computed(() => [
  { value: "neutral", label: t("settings.theme.fontStyle.neutral") },
  { value: "reading", label: t("settings.theme.fontStyle.reading") },
  { value: "coding", label: t("settings.theme.fontStyle.coding") }
]);

const fontScaleOptions = computed(() => [
  { value: "sm", label: t("settings.theme.fontScale.sm") },
  { value: "md", label: t("settings.theme.fontScale.md") },
  { value: "lg", label: t("settings.theme.fontScale.lg") }
]);

const presetOptions = computed(() => [
  { value: "aurora_forge", label: t("settings.theme.preset.aurora_forge") },
  { value: "obsidian_pulse", label: t("settings.theme.preset.obsidian_pulse") },
  { value: "paper_focus", label: t("settings.theme.preset.paper_focus") }
]);
</script>

<style scoped>
.theme-layout {
  display: grid;
  grid-template-columns: minmax(520px, 920px);
  gap: var(--global-space-12);
  min-height: 0;
}

.config-panel {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  padding: var(--global-space-12);
  display: grid;
  gap: var(--global-space-12);
  align-content: start;
}

.panel-header {
  display: flex;
  justify-content: flex-end;
}

.config-panel {
  grid-template-rows: auto repeat(4, auto);
}

.config-group {
  display: grid;
  gap: var(--global-space-8);
  padding-bottom: var(--global-space-12);
  border-bottom: 1px solid var(--semantic-divider);
}

.group-header {
  display: grid;
  gap: var(--global-space-4);
}

.group-header h4,
.group-header p {
  margin: 0;
}

.group-header h4 {
  color: var(--semantic-text);
  font-size: var(--global-font-size-13);
  font-weight: var(--global-font-weight-600);
}

.group-header p {
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
}

@media (max-width: 1140px) {
  .theme-layout {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
