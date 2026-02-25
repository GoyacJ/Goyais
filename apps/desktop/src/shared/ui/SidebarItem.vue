<template>
  <button
    class="w-full border border-transparent rounded-[var(--global-radius-8)] bg-transparent px-[var(--global-space-12)] py-[var(--global-space-8)] text-left [font-family:var(--global-font-family-ui)] text-[var(--global-font-size-13)] text-[var(--component-sidebar-item-fg)] hover:bg-[var(--component-sidebar-item-bg-hover)]"
    :class="{
      'bg-[var(--component-sidebar-item-bg-active)] text-[var(--component-sidebar-item-fg-active)]': active,
      'opacity-[var(--component-tree-item-disabled-opacity)] cursor-not-allowed': disabled
    }"
    :disabled="disabled"
    type="button"
  >
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
