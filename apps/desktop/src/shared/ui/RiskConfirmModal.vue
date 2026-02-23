<template>
  <BaseModal :open="open">
    <template #title>
      <strong>{{ title }}</strong>
    </template>
    <div class="risk-box">
      <p>{{ summary }}</p>
      <pre>{{ preview }}</pre>
    </div>
    <template #footer>
      <slot />
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import BaseModal from "@/shared/ui/BaseModal.vue";

defineProps<{
  open: boolean;
  title: string;
  summary: string;
  preview: string;
}>();
</script>

<style scoped>
.risk-box {
  background: var(--component-risk-confirm-bg);
  border: 1px solid var(--component-risk-confirm-border);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-12);
  display: grid;
  gap: var(--global-space-8);
}

pre {
  margin: 0;
  background: var(--component-risk-confirm-command-bg);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8);
  white-space: pre-wrap;
}

p {
  margin: 0;
}
</style>
