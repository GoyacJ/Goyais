<template>
  <ConfigSectionCard
    v-for="card in cards"
    :key="card.title"
    :title="card.title"
    :lines="card.lines"
    :tone="card.tone"
  />
</template>

<script setup lang="ts">
import ConfigSectionCard from "@/shared/ui/ConfigSectionCard.vue";

defineProps<{
  cards: Array<{
    title: string;
    lines: string[];
    tone: "default" | "info" | "warning" | "danger" | "success";
  }>;
}>();
</script>
