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
    <section class="theme-layout grid min-h-0 gap-[var(--global-space-12)] [grid-template-columns:minmax(520px,920px)] max-[1140px]:[grid-template-columns:minmax(0,1fr)]">
      <article
        class="settings-panel grid content-start gap-[var(--global-space-12)] border border-[var(--semantic-border)] rounded-[var(--global-radius-12)] bg-[var(--semantic-surface)] p-[var(--global-space-12)]"
      >
        <header class="panel-header grid gap-[var(--global-space-4)]">
          <h3 class="m-0">{{ t("settings.i18n.title") }}</h3>
        </header>

        <section class="config-group grid gap-[var(--global-space-8)] border-b border-[var(--semantic-divider)] pb-[var(--global-space-12)]">
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
