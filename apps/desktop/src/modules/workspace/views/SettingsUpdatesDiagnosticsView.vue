<template>
  <SettingsShell
    active-key="settings_updates_diagnostics"
    title="更新与诊断"
    subtitle="Local Settings / Updates & Diagnostics"
  >
    <LocalSettingsPage :cards="localUpdatesCards" />
  </SettingsShell>
</template>

<script setup lang="ts">
import LocalSettingsPage from "@/modules/workspace/components/LocalSettingsPage.vue";
import { localUpdatesCards } from "@/modules/workspace/schemas/settingsContent";
import SettingsShell from "@/shared/shells/SettingsShell.vue";
</script>
