<template>
  <BaseCard class="gap-[var(--global-space-8)]">
    <template #header>
      <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-[var(--global-space-8)]">
        <div class="min-w-0 text-[var(--global-font-size-13)] text-[var(--semantic-text)]">
          <slot name="title" />
        </div>
        <div v-if="$slots.status">
          <slot name="status" />
        </div>
      </div>
    </template>

    <div class="grid gap-[var(--global-space-4)]">
      <slot />
    </div>

    <details
      v-if="$slots.details"
      class="border border-dashed border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-8)]"
    >
      <summary class="cursor-pointer text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">{{ detailsLabel }}</summary>
      <div class="mt-[var(--global-space-8)] grid gap-[var(--global-space-4)]">
        <slot name="details" />
      </div>
    </details>

    <template v-if="hasActions" #footer>
      <div class="grid gap-[var(--global-space-8)]">
        <div v-if="$slots.actionsPrimary" class="actions-primary flex flex-wrap gap-[var(--global-space-8)]">
          <slot name="actionsPrimary" />
        </div>
        <div v-if="$slots.actionsSecondary" class="actions-secondary flex flex-wrap gap-[var(--global-space-8)]">
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
