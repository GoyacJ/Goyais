<template>
  <SettingsShell
    active-key="settings_i18n"
    title="国际化"
    subtitle="Local Settings / i18n"
  >
    <LocalSettingsPage :cards="localI18nCards" />
  </SettingsShell>
</template>

<script setup lang="ts">
import LocalSettingsPage from "@/modules/workspace/components/LocalSettingsPage.vue";
import { localI18nCards } from "@/modules/workspace/schemas/settingsContent";
import SettingsShell from "@/shared/shells/SettingsShell.vue";
</script>
