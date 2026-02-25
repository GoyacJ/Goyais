<template>
  <div
    class="border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] px-[var(--global-space-12)] py-[var(--global-space-8)] text-[var(--global-font-size-12)]"
    :class="toneClassMap[tone]"
  >
    <slot>{{ message }}</slot>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  tone: "error" | "warning" | "info" | "403" | "disconnected" | "retrying";
  message?: string;
}>();

const toneClassMap = {
  error: "bg-[var(--component-toast-error-bg)] text-[var(--component-toast-error-fg)]",
  "403": "bg-[var(--component-toast-error-bg)] text-[var(--component-toast-error-fg)]",
  warning: "bg-[var(--component-toast-warning-bg)] text-[var(--component-toast-warning-fg)]",
  disconnected: "bg-[var(--component-toast-warning-bg)] text-[var(--component-toast-warning-fg)]",
  info: "bg-[var(--component-toast-info-bg)] text-[var(--component-toast-info-fg)]",
  retrying: "bg-[var(--component-toast-info-bg)] text-[var(--component-toast-info-fg)]"
} as const;
</script>
