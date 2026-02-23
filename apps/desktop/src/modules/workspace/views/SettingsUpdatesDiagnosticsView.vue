<template>
  <LocalSettingsPage
    active-key="settings_updates_diagnostics"
    title="更新与诊断"
    subtitle="Local Settings / Updates & Diagnostics"
    :menu-entries="menuEntries"
    :cards="localUpdatesCards"
  />
</template>

<script setup lang="ts">
import LocalSettingsPage from "@/modules/workspace/components/LocalSettingsPage.vue";
import { localUpdatesCards } from "@/modules/workspace/schemas/settingsContent";
import { useLocalSettingsMenu } from "@/shared/navigation/pageMenus";

const menuEntries = useLocalSettingsMenu();
</script>
