<template>
  <button class="sidebar-item" :class="{ active, disabled }" :disabled="disabled" type="button">
    <slot />
  </button>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    active?: boolean;
    disabled?: boolean;
  }>(),
  {
    active: false,
    disabled: false
  }
);
</script>

<style scoped>
.sidebar-item {
  width: 100%;
  border: 1px solid transparent;
  border-radius: var(--global-radius-8);
  background: transparent;
  color: var(--component-sidebar-item-fg);
  text-align: left;
  padding: var(--global-space-8) var(--global-space-12);
  font-family: var(--global-font-family-ui);
  font-size: var(--global-font-size-13);
}

.sidebar-item:hover {
  background: var(--component-sidebar-item-bg-hover);
}

.sidebar-item.active {
  background: var(--component-sidebar-item-bg-active);
  color: var(--component-sidebar-item-fg-active);
}

.sidebar-item.disabled {
  opacity: var(--component-tree-item-disabled-opacity);
  cursor: not-allowed;
}
</style>
