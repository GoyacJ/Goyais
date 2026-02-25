<template>
  <div
    class="grid grid-flow-col items-center gap-[var(--global-space-12)] border-b border-[var(--component-table-row-divider)] px-[var(--global-space-12)] py-[var(--global-space-8)] text-[var(--component-table-row-fg)] hover:bg-[var(--component-table-row-bg-hover)]"
    :class="{ 'bg-[var(--component-table-row-bg-selected)]': selected }"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    selected?: boolean;
  }>(),
  {
    selected: false
  }
);
</script>
