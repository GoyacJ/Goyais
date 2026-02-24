<template>
  <div class="table-row" :class="{ selected }">
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

<style scoped>
.table-row {
  display: grid;
  grid-auto-flow: column;
  gap: var(--global-space-12);
  align-items: center;
  padding: var(--global-space-8) var(--global-space-12);
  border-bottom: 1px solid var(--component-table-row-divider);
  color: var(--component-table-row-fg);
}

.table-row:hover {
  background: var(--component-table-row-bg-hover);
}

.table-row.selected {
  background: var(--component-table-row-bg-selected);
}
</style>
