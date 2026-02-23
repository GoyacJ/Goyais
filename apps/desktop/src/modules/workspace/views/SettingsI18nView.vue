<template>
  <LocalSettingsPage
    active-key="settings_i18n"
    title="国际化"
    subtitle="Local Settings / i18n"
    :menu-entries="menuEntries"
    :cards="localI18nCards"
  />
</template>

<script setup lang="ts">
import LocalSettingsPage from "@/modules/workspace/components/LocalSettingsPage.vue";
import { localI18nCards } from "@/modules/workspace/schemas/settingsContent";
import { useLocalSettingsMenu } from "@/shared/navigation/pageMenus";

const menuEntries = useLocalSettingsMenu();
</script>
