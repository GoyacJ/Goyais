<template>
  <UnifiedConfigSidebar variant="remote" :active-key="activeKey" :menu-entries="menuEntries" />
</template>

<script setup lang="ts">
import type { MenuEntry } from "@/shared/navigation/pageMenus";
import UnifiedConfigSidebar from "@/shared/layouts/UnifiedConfigSidebar.vue";

defineProps<{
  activeKey: string;
  scopeHint: string;
  menuEntries: MenuEntry[];
}>();
</script>
