<template>
  <div class="list-item" :class="{ active }">
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

<style scoped>
.list-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--global-space-8);
  color: var(--component-list-item-fg);
  border: 1px solid var(--component-list-item-border);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8) var(--global-space-12);
}

.list-item:hover {
  background: var(--component-list-item-bg-hover);
}

.list-item.active {
  background: var(--component-list-item-bg-active);
}
</style>
