<template>
  <footer class="status-bar">
    <span class="left">
      <AppIcon name="plug-zap" :size="12" />
      Hub: {{ hubLabel }}
    </span>
    <span class="right">
      <span>{{ roleLabel }}</span>
      <span class="dot" :class="connectionClass"></span>
      <span :class="connectionClass">{{ connectionLabel }}</span>
    </span>
  </footer>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { authStore } from "@/shared/stores/authStore";
import { getCurrentWorkspace, workspaceStore } from "@/shared/stores/workspaceStore";
import AppIcon from "@/shared/ui/AppIcon.vue";

const connectionLabel = computed(() => {
  if (workspaceStore.connectionState === "ready") {
    return "connected";
  }
  if (workspaceStore.connectionState === "loading") {
    return "reconnecting";
  }
  return "disconnected";
});

const connectionClass = computed(() => {
  if (connectionLabel.value === "connected") {
    return "connected";
  }
  if (connectionLabel.value === "reconnecting") {
    return "reconnecting";
  }
  return "disconnected";
});

const hubLabel = computed(() => {
  return getCurrentWorkspace()?.hub_url ?? "local://workspace";
});

const roleLabel = computed(() => authStore.me?.role ?? "Owner");
</script>

<style scoped>
.status-bar {
  height: 36px;
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  color: var(--semantic-text-muted);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--global-space-12);
  font-size: var(--global-font-size-11);
  gap: var(--global-space-12);
}

.left,
.right {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--semantic-text-subtle);
}

.connected {
  color: var(--semantic-success);
}

.reconnecting {
  color: var(--semantic-warning);
}

.disconnected {
  color: var(--semantic-danger);
}
</style>
