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
    <section class="theme-layout grid min-h-0 gap-[var(--global-space-12)] [grid-template-columns:minmax(520px,920px)] max-[1140px]:[grid-template-columns:minmax(0,1fr)]">
      <article
        class="config-panel grid content-start grid-rows-[auto_repeat(4,auto)] gap-[var(--global-space-12)] border border-[var(--semantic-border)] rounded-[var(--global-radius-12)] bg-[var(--semantic-surface)] p-[var(--global-space-12)]"
      >
        <header class="panel-header flex justify-end">
          <BaseButton data-testid="theme-reset-button" variant="secondary" @click="theme.resetThemeSettings()">
            {{ t("settings.theme.reset") }}
          </BaseButton>
        </header>

        <section class="config-group grid gap-[var(--global-space-8)] border-b border-[var(--semantic-divider)] pb-[var(--global-space-12)]">
          <div class="group-header grid gap-[var(--global-space-4)]">
            <h4 class="m-0 text-[var(--global-font-size-13)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
              {{ t("settings.theme.modeTitle") }}
            </h4>
            <p class="m-0 text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">{{ t("settings.theme.modeHint") }}</p>
          </div>
          <BaseSelect data-testid="theme-mode-select" v-model="themeModeModel" :options="themeModeOptions" />
        </section>

        <section class="config-group grid gap-[var(--global-space-8)] border-b border-[var(--semantic-divider)] pb-[var(--global-space-12)]">
          <div class="group-header grid gap-[var(--global-space-4)]">
            <h4 class="m-0 text-[var(--global-font-size-13)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
              {{ t("settings.theme.fontStyleTitle") }}
            </h4>
            <p class="m-0 text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">{{ t("settings.theme.fontStyleHint") }}</p>
          </div>
          <BaseSelect data-testid="font-style-select" v-model="fontStyleModel" :options="fontStyleOptions" />
        </section>

        <section class="config-group grid gap-[var(--global-space-8)] border-b border-[var(--semantic-divider)] pb-[var(--global-space-12)]">
          <div class="group-header grid gap-[var(--global-space-4)]">
            <h4 class="m-0 text-[var(--global-font-size-13)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
              {{ t("settings.theme.fontScaleTitle") }}
            </h4>
            <p class="m-0 text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">{{ t("settings.theme.fontScaleHint") }}</p>
          </div>
          <BaseSelect data-testid="font-scale-select" v-model="fontScaleModel" :options="fontScaleOptions" />
        </section>

        <section class="config-group grid gap-[var(--global-space-8)] border-b border-[var(--semantic-divider)] pb-[var(--global-space-12)]">
          <div class="group-header grid gap-[var(--global-space-4)]">
            <h4 class="m-0 text-[var(--global-font-size-13)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
              {{ t("settings.theme.presetTitle") }}
            </h4>
            <p class="m-0 text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">{{ t("settings.theme.presetHint") }}</p>
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
