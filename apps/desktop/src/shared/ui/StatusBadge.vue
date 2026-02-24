<template>
  <span class="badge" :class="`tone-${tone}`">
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
</script>

<style scoped>
.badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--component-badge-radius);
  padding: var(--global-space-2px) var(--global-space-8);
  font-size: var(--component-badge-font-size);
  font-family: var(--global-font-family-ui);
}

.tone-queued {
  background: var(--component-badge-execution-queued-bg);
  color: var(--component-badge-execution-queued-fg);
}

.tone-running {
  background: var(--component-badge-execution-running-bg);
  color: var(--component-badge-execution-running-fg);
}

.tone-success {
  background: var(--component-badge-execution-success-bg);
  color: var(--component-badge-execution-success-fg);
}

.tone-failed {
  background: var(--component-badge-execution-failed-bg);
  color: var(--component-badge-execution-failed-fg);
}

.tone-cancelled {
  background: var(--component-badge-execution-cancelled-bg);
  color: var(--component-badge-execution-cancelled-fg);
}

.tone-connected {
  background: var(--component-badge-network-connected-bg);
  color: var(--component-badge-network-connected-fg);
}

.tone-disconnected {
  background: var(--component-badge-network-disconnected-bg);
  color: var(--component-badge-network-disconnected-fg);
}
</style>
