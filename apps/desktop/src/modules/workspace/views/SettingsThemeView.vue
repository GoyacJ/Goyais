<template>
  <SettingsShell
    active-key="settings_theme"
    :title="t('menu.settingsTheme')"
    :subtitle="`${t('settings.scope')} / Theme`"
  >
    <section class="card">
      <h3>{{ t("settings.theme.title") }}</h3>
      <BaseSelect v-model="themePreference" :options="themeOptions" />
      <p class="hint">{{ t("settings.theme.current") }}: {{ resolvedLabel }}</p>
    </section>

    <section class="card">
      <h3>{{ t("settings.theme.preview") }}</h3>
      <p class="hint">{{ t("settings.theme.previewHint") }}</p>
    </section>
  </SettingsShell>
</template>

<script setup lang="ts">
import { computed } from "vue";

import SettingsShell from "@/shared/shells/SettingsShell.vue";
import { useI18n } from "@/shared/i18n";
import BaseSelect from "@/shared/ui/BaseSelect.vue";
import { useTheme, type ThemePreference } from "@/shared/stores/themeStore";

const { t } = useI18n();
const theme = useTheme();

const themePreference = computed<ThemePreference>({
  get: () => theme.preference.value,
  set: (value) => theme.setThemePreference(value)
});

const themeOptions = computed(() => [
  { value: "system", label: t("settings.theme.system") },
  { value: "dark", label: t("settings.theme.dark") },
  { value: "light", label: t("settings.theme.light") }
]);

const resolvedLabel = computed(() => {
  if (theme.resolved.value === "light") {
    return t("settings.theme.light");
  }
  return t("settings.theme.dark");
});
</script>

<style scoped>
.card {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  padding: var(--global-space-12);
  display: grid;
  gap: var(--global-space-8);
}

.card h3,
.hint {
  margin: 0;
}

.hint {
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
}
</style>
