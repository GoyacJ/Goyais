<template>
  <div
    class="flex items-center justify-between gap-[var(--global-space-8)] border border-[var(--component-list-item-border)] rounded-[var(--global-radius-8)] px-[var(--global-space-12)] py-[var(--global-space-8)] text-[var(--component-list-item-fg)] hover:bg-[var(--component-list-item-bg-hover)]"
    :class="{ 'bg-[var(--component-list-item-bg-active)]': active }"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    active?: boolean;
  }>(),
  {
    active: false
  }
);
</script>
