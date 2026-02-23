<template>
  <LocalSettingsPage
    active-key="settings_general"
    title="通用设置"
    subtitle="Local Settings / General"
    :menu-entries="menuEntries"
    :cards="localGeneralCards"
  />
</template>

<script setup lang="ts">
import LocalSettingsPage from "@/modules/workspace/components/LocalSettingsPage.vue";
import { localGeneralCards } from "@/modules/workspace/schemas/settingsContent";
import { useLocalSettingsMenu } from "@/shared/navigation/pageMenus";

const menuEntries = useLocalSettingsMenu();
</script>
