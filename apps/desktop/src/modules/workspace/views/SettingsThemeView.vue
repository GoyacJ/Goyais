<template>
  <SettingsShell
    active-key="settings_theme"
    title="主题"
    subtitle="Local Settings / Theme"
  >
    <LocalSettingsPage :cards="localThemeCards" />
  </SettingsShell>
</template>

<script setup lang="ts">
import LocalSettingsPage from "@/modules/workspace/components/LocalSettingsPage.vue";
import { localThemeCards } from "@/modules/workspace/schemas/settingsContent";
import SettingsShell from "@/shared/shells/SettingsShell.vue";
</script>
