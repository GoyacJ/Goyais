<template>
  <LocalSettingsLayout
    :active-key="activeKey"
    :menu-entries="menuEntries"
    :title="title"
    :subtitle="subtitle"
  >
    <ConfigSectionCard
      v-for="card in cards"
      :key="card.title"
      :title="card.title"
      :lines="card.lines"
      :tone="card.tone"
    />
  </LocalSettingsLayout>
</template>

<script setup lang="ts">
import type { MenuEntry } from "@/shared/navigation/pageMenus";
import LocalSettingsLayout from "@/shared/layouts/LocalSettingsLayout.vue";
import ConfigSectionCard from "@/shared/ui/ConfigSectionCard.vue";

defineProps<{
  activeKey: string;
  title: string;
  subtitle: string;
  menuEntries: MenuEntry[];
  cards: Array<{
    title: string;
    lines: string[];
    tone: "default" | "info" | "warning" | "danger" | "success";
  }>;
}>();
</script>
