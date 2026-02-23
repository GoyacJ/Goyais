<template>
  <div class="toast" :class="`tone-${tone}`">
    <slot>{{ message }}</slot>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  tone: "error" | "warning" | "info" | "403" | "disconnected" | "retrying";
  message?: string;
}>();
</script>

<style scoped>
.toast {
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8) var(--global-space-12);
  border: 1px solid var(--semantic-border);
  font-size: var(--global-font-size-12);
}

.tone-error,
.tone-403 {
  background: var(--component-toast-error-bg);
  color: var(--component-toast-error-fg);
}

.tone-warning,
.tone-disconnected {
  background: var(--component-toast-warning-bg);
  color: var(--component-toast-warning-fg);
}

.tone-info,
.tone-retrying {
  background: var(--component-toast-info-bg);
  color: var(--component-toast-info-fg);
}
</style>
