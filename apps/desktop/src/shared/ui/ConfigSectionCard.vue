<template>
  <BaseCard>
    <template #header>
      <strong>{{ title }}</strong>
    </template>

    <div class="lines" :class="`tone-${tone}`">
      <p v-for="line in lines" :key="line" :class="{ mono }">{{ line }}</p>
    </div>
  </BaseCard>
</template>

<script setup lang="ts">
import BaseCard from "@/shared/ui/BaseCard.vue";

withDefaults(
  defineProps<{
    title: string;
    lines: string[];
    tone?: "default" | "info" | "warning" | "danger" | "success";
    mono?: boolean;
  }>(),
  {
    tone: "default",
    mono: false
  }
);
</script>

<style scoped>
.lines {
  display: grid;
  gap: var(--global-space-8);
}
p {
  margin: 0;
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
  line-height: var(--global-line-height-normal);
}
p.mono {
  font-family: ui-monospace, "SF Mono", Menlo, monospace;
  white-space: pre-wrap;
}
.tone-info p {
  color: var(--component-toast-info-fg);
}
.tone-warning p {
  color: var(--component-toast-warning-fg);
}
.tone-danger p {
  color: var(--semantic-danger);
}
.tone-success p {
  color: var(--semantic-success);
}
</style>
