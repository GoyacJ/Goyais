<template>
  <BaseModal :open="open">
    <template #title>
      <h3 class="title">{{ title }}</h3>
    </template>

    <div class="body">
      <pre>{{ formatted }}</pre>
    </div>

    <template #footer>
      <button type="button" class="action" @click="emit('close')">关闭</button>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed } from "vue";

import BaseModal from "@/shared/ui/BaseModal.vue";

const props = withDefaults(
  defineProps<{
    open: boolean;
    payload: Record<string, unknown> | null;
    title?: string;
  }>(),
  {
    title: "MCP 聚合 JSON"
  }
);

const emit = defineEmits<{
  (event: "close"): void;
}>();

const formatted = computed(() => JSON.stringify(props.payload ?? {}, null, 2));
</script>

<style scoped>
.title {
  margin: 0;
}

.body {
  max-height: min(60vh, 520px);
  overflow: auto;
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  padding: var(--global-space-8);
}

pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
}

.action {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface-2);
  color: var(--semantic-text);
  padding: var(--global-space-6) var(--global-space-10);
}
</style>
