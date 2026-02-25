<template>
  <button
    :class="[
      'h-[var(--component-button-height)] rounded-[var(--component-button-radius)] px-[var(--component-button-padding-x)] py-[var(--component-button-padding-y)] text-[var(--component-button-font-size)] [font-family:var(--global-font-family-ui)] border border-transparent inline-flex items-center justify-center gap-[var(--global-space-8)] cursor-pointer disabled:(bg-[var(--component-button-disabled-bg)] border-[var(--component-button-disabled-border)] text-[var(--component-button-disabled-fg)] cursor-not-allowed)',
      variantClassMap[variant],
      `variant-${variant}`,
      loading ? 'opacity-[var(--component-button-loading-opacity)]' : '',
      iconOnly ? 'w-[var(--component-button-height)] p-0' : ''
    ]"
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

const variantClassMap = {
  primary: "bg-[var(--component-button-primary-bg)] border-[var(--component-button-primary-border)] text-[var(--component-button-primary-fg)]",
  secondary:
    "bg-[var(--component-button-secondary-bg)] border-[var(--component-button-secondary-border)] text-[var(--component-button-secondary-fg)]",
  ghost: "bg-[var(--component-button-ghost-bg)] border-[var(--component-button-ghost-border)] text-[var(--component-button-ghost-fg)]",
  danger: "bg-[var(--component-button-danger-bg)] border-[var(--component-button-danger-border)] text-[var(--component-button-danger-fg)]"
} as const;

function onClick(event: MouseEvent): void {
  emit("click", event);
}
</script>
