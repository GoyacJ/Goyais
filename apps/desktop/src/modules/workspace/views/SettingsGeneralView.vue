<template>
  <SettingsShell
    active-key="settings_general"
    title="通用设置"
    subtitle="Local Settings / General"
  >
    <LocalSettingsPage :cards="localGeneralCards" />
  </SettingsShell>
</template>

<script setup lang="ts">
import LocalSettingsPage from "@/modules/workspace/components/LocalSettingsPage.vue";
import { localGeneralCards } from "@/modules/workspace/schemas/settingsContent";
import SettingsShell from "@/shared/shells/SettingsShell.vue";
</script>
