<template>
  <button
    :class="['ui-button', `variant-${variant}`, { loading, iconOnly }]"
    :disabled="disabled || loading"
    type="button"
    @click="onClick"
  >
    <slot />
  </button>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    variant?: "primary" | "secondary" | "ghost" | "danger";
    disabled?: boolean;
    loading?: boolean;
    iconOnly?: boolean;
  }>(),
  {
    variant: "secondary",
    disabled: false,
    loading: false,
    iconOnly: false
  }
);

const emit = defineEmits<{
  (event: "click", payload: MouseEvent): void;
}>();

function onClick(event: MouseEvent): void {
  emit("click", event);
}
</script>

<style scoped>
.ui-button {
  height: var(--component-button-height);
  border-radius: var(--component-button-radius);
  padding: var(--component-button-padding-y) var(--component-button-padding-x);
  font-size: var(--component-button-font-size);
  font-family: var(--global-font-family-ui);
  border: 1px solid transparent;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--global-space-8);
  cursor: pointer;
}

.ui-button.iconOnly {
  width: var(--component-button-height);
  padding: 0;
}

.ui-button.variant-primary {
  background: var(--component-button-primary-bg);
  border-color: var(--component-button-primary-border);
  color: var(--component-button-primary-fg);
}

.ui-button.variant-secondary {
  background: var(--component-button-secondary-bg);
  border-color: var(--component-button-secondary-border);
  color: var(--component-button-secondary-fg);
}

.ui-button.variant-ghost {
  background: var(--component-button-ghost-bg);
  border-color: var(--component-button-ghost-border);
  color: var(--component-button-ghost-fg);
}

.ui-button.variant-danger {
  background: var(--component-button-danger-bg);
  border-color: var(--component-button-danger-border);
  color: var(--component-button-danger-fg);
}

.ui-button:disabled {
  background: var(--component-button-disabled-bg);
  border-color: var(--component-button-disabled-border);
  color: var(--component-button-disabled-fg);
  cursor: not-allowed;
}

.ui-button.loading {
  opacity: var(--component-button-loading-opacity);
}
</style>
