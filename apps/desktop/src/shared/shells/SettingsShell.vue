<template>
  <LocalSettingsLayout
    :active-key="activeKey"
    :menu-entries="resolvedMenuEntries"
    :title="title"
    :subtitle="subtitle"
  >
    <slot />
  </LocalSettingsLayout>
</template>

<script setup lang="ts">
import { computed } from "vue";

import LocalSettingsLayout from "@/shared/layouts/LocalSettingsLayout.vue";
import type { MenuEntry } from "@/shared/navigation/pageMenus";
import { useLocalSettingsMenu } from "@/shared/navigation/pageMenus";

const props = defineProps<{
  activeKey: string;
  title: string;
  subtitle: string;
  menuEntries?: MenuEntry[];
}>();

const menuEntries = useLocalSettingsMenu();
const resolvedMenuEntries = computed(() => props.menuEntries ?? menuEntries.value);
</script>
