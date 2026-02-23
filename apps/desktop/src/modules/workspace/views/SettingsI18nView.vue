<template>
  <SettingsShell
    active-key="settings_i18n"
    :title="t('menu.settingsI18n')"
    :subtitle="`${t('settings.scope')} / i18n`"
  >
    <section class="card">
      <h3>{{ t("settings.i18n.title") }}</h3>
      <BaseSelect v-model="localeModel" :options="localeOptions" />
      <p class="hint">{{ t("settings.i18n.current") }}: {{ localeModel }}</p>
    </section>

    <section class="card">
      <h3>{{ t("settings.i18n.preview") }}</h3>
      <p class="hint">{{ t("settings.i18n.previewHint") }}</p>
      <p class="hint">{{ t("conversation.placeholderInput") }}</p>
    </section>
  </SettingsShell>
</template>

<script setup lang="ts">
import { computed } from "vue";

import SettingsShell from "@/shared/shells/SettingsShell.vue";
import { availableLocales, setLocale, useI18n } from "@/shared/i18n";
import type { Locale } from "@/shared/i18n/messages";
import BaseSelect from "@/shared/ui/BaseSelect.vue";

const { locale, t } = useI18n();

const localeOptions = computed(() =>
  availableLocales.map((item) => ({
    value: item,
    label: item
  }))
);

const localeModel = computed<string>({
  get: () => locale.value,
  set: (value) => setLocale(value as Locale)
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
