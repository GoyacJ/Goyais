<template>
  <SettingsShell
    active-key="settings_i18n"
    :title="t('menu.settingsI18n')"
    :subtitle="`${t('settings.scope')} / i18n`"
    runtime-status-mode
    :runtime-conversation-status="workspaceStatus.conversationStatus.value"
    :runtime-connection-status="workspaceStatus.connectionStatus.value"
    :runtime-user-display-name="workspaceStatus.userDisplayName.value"
    :runtime-hub-url="workspaceStatus.hubURL.value"
  >
    <section class="theme-layout">
      <article class="settings-panel">
        <header class="panel-header">
          <h3>{{ t("settings.i18n.title") }}</h3>
        </header>

        <section class="config-group">
          <BaseSelect data-testid="locale-select" v-model="localeModel" :options="localeOptions" />
        </section>
      </article>
    </section>
  </SettingsShell>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { projectStore } from "@/modules/project/store";
import SettingsShell from "@/shared/shells/SettingsShell.vue";
import { availableLocales, setLocale, useI18n } from "@/shared/i18n";
import type { Locale } from "@/shared/i18n/messages";
import { useWorkspaceStatusSync } from "@/shared/stores/workspaceStatusStore";
import BaseSelect from "@/shared/ui/BaseSelect.vue";

const { locale, t } = useI18n();
const workspaceStatus = useWorkspaceStatusSync({
  conversationId: computed(() => projectStore.activeConversationId)
});

const localeLabelKeyMap: Record<Locale, string> = {
  "zh-CN": "settings.i18n.option.zhCN",
  "en-US": "settings.i18n.option.enUS"
};

const localeOptions = computed(() =>
  availableLocales.map((item) => ({
    value: item,
    label: `${t(localeLabelKeyMap[item])}（${item}）`
  }))
);

const localeModel = computed<string>({
  get: () => locale.value,
  set: (value) => setLocale(value as Locale)
});
</script>

<style scoped>
.theme-layout {
  display: grid;
  grid-template-columns: minmax(520px, 920px);
  gap: var(--global-space-12);
  min-height: 0;
}

.settings-panel {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  padding: var(--global-space-12);
  display: grid;
  gap: var(--global-space-12);
  align-content: start;
}

.panel-header {
  display: grid;
  gap: var(--global-space-4);
}

.config-group {
  display: grid;
  gap: var(--global-space-8);
  padding-bottom: var(--global-space-12);
  border-bottom: 1px solid var(--semantic-divider);
}

@media (max-width: 1140px) {
  .theme-layout {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
