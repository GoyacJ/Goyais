<template>
  <WorkspaceSharedShell
    :active-key="activeKey"
    :title="title"
    :account-subtitle="accountSubtitle"
    :settings-subtitle="settingsSubtitle"
  >
    <ConfigSectionCard
      v-for="card in cards"
      :key="card.title"
      :title="card.title"
      :lines="card.lines"
      :tone="card.tone"
      :mono="card.mono"
    />
  </WorkspaceSharedShell>
</template>

<script setup lang="ts">
import WorkspaceSharedShell from "@/shared/shells/WorkspaceSharedShell.vue";
import ConfigSectionCard from "@/shared/ui/ConfigSectionCard.vue";

type CardSpec = {
  title: string;
  lines: string[];
  tone: "default" | "info" | "warning" | "danger" | "success";
  mono?: boolean;
};

const props = defineProps<{
  activeKey: string;
  title: string;
  accountSubtitle: string;
  settingsSubtitle: string;
  cards: CardSpec[];
}>();
</script>
