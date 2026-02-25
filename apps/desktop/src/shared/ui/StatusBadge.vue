<template>
  <span
    class="inline-flex items-center justify-center rounded-[var(--component-badge-radius)] px-[var(--global-space-8)] py-[var(--global-space-2px)] [font-family:var(--global-font-family-ui)] text-[var(--component-badge-font-size)]"
    :class="toneClassMap[tone]"
  >
    <slot>{{ label }}</slot>
  </span>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    label?: string;
    tone:
      | "queued"
      | "running"
      | "success"
      | "failed"
      | "cancelled"
      | "connected"
      | "disconnected";
  }>(),
  {
    label: ""
  }
);

const toneClassMap = {
  queued: "bg-[var(--component-badge-execution-queued-bg)] text-[var(--component-badge-execution-queued-fg)]",
  running: "bg-[var(--component-badge-execution-running-bg)] text-[var(--component-badge-execution-running-fg)]",
  success: "bg-[var(--component-badge-execution-success-bg)] text-[var(--component-badge-execution-success-fg)]",
  failed: "bg-[var(--component-badge-execution-failed-bg)] text-[var(--component-badge-execution-failed-fg)]",
  cancelled: "bg-[var(--component-badge-execution-cancelled-bg)] text-[var(--component-badge-execution-cancelled-fg)]",
  connected: "bg-[var(--component-badge-network-connected-bg)] text-[var(--component-badge-network-connected-fg)]",
  disconnected: "bg-[var(--component-badge-network-disconnected-bg)] text-[var(--component-badge-network-disconnected-fg)]"
} as const;
</script>
