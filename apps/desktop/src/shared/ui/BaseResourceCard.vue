<template>
  <BaseCard class="resource-card">
    <template #header>
      <div class="card-header">
        <div class="title">
          <slot name="title" />
        </div>
        <div v-if="$slots.status" class="status">
          <slot name="status" />
        </div>
      </div>
    </template>

    <div class="summary">
      <slot />
    </div>

    <details v-if="$slots.details" class="details">
      <summary>{{ detailsLabel }}</summary>
      <div class="details-content">
        <slot name="details" />
      </div>
    </details>

    <template v-if="hasActions" #footer>
      <div class="actions">
        <div v-if="$slots.actionsPrimary" class="actions-primary">
          <slot name="actionsPrimary" />
        </div>
        <div v-if="$slots.actionsSecondary" class="actions-secondary">
          <slot name="actionsSecondary" />
        </div>
      </div>
    </template>
  </BaseCard>
</template>

<script setup lang="ts">
import { computed, useSlots } from "vue";

import BaseCard from "@/shared/ui/BaseCard.vue";

const slots = useSlots();
const hasActions = computed(() => slots.actionsPrimary !== undefined || slots.actionsSecondary !== undefined);

withDefaults(
  defineProps<{
    detailsLabel?: string;
  }>(),
  {
    detailsLabel: "查看详情"
  }
);
</script>

<style scoped>
.resource-card {
  gap: var(--global-space-8);
}

.card-header {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: var(--global-space-8);
  align-items: center;
}

.title {
  min-width: 0;
  font-size: var(--global-font-size-13);
  color: var(--semantic-text);
}

.summary {
  display: grid;
  gap: var(--global-space-4);
}

.details {
  border: 1px dashed var(--semantic-border);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8);
  background: var(--semantic-bg);
}

.details summary {
  cursor: pointer;
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
}

.details-content {
  margin-top: var(--global-space-8);
  display: grid;
  gap: var(--global-space-4);
}

.actions {
  display: grid;
  gap: var(--global-space-8);
}

.actions-primary,
.actions-secondary {
  display: flex;
  flex-wrap: wrap;
  gap: var(--global-space-8);
}
</style>
