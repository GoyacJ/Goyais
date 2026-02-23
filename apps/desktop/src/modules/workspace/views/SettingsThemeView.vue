<template>
  <LocalSettingsPage
    active-key="settings_theme"
    title="主题"
    subtitle="Local Settings / Theme"
    :menu-entries="menuEntries"
    :cards="localThemeCards"
  />
</template>

<script setup lang="ts">
import LocalSettingsPage from "@/modules/workspace/components/LocalSettingsPage.vue";
import { localThemeCards } from "@/modules/workspace/schemas/settingsContent";
import { useLocalSettingsMenu } from "@/shared/navigation/pageMenus";

const menuEntries = useLocalSettingsMenu();
</script>
