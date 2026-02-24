<template>
  <footer class="status-bar">
    <span class="left">
      <AppIcon name="plug-zap" :size="12" />
      Hub: {{ hubLabel }}
    </span>
    <span class="right">
      <span>{{ identityLabel }}</span>
      <span class="dot" :class="connectionClass"></span>
      <span :class="connectionClass">{{ connectionLabel }}</span>
    </span>
  </footer>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { authStore } from "@/shared/stores/authStore";
import { getCurrentWorkspace, workspaceStore, type ConnectionState } from "@/shared/stores/workspaceStore";
import type { ConnectionStatus } from "@/shared/types/api";
import AppIcon from "@/shared/ui/AppIcon.vue";

const props = withDefaults(
  defineProps<{
    runtimeMode?: boolean;
    hubLabel?: string;
    roleLabel?: string;
    userLabel?: string;
    connectionState?: ConnectionState;
    connectionStatus?: ConnectionStatus;
  }>(),
  {
    runtimeMode: false
  }
);

const resolvedConnectionState = computed(() => props.connectionState ?? workspaceStore.connectionState);

const connectionLabel = computed<ConnectionStatus>(() => {
  if (props.connectionStatus === "connected" || props.connectionStatus === "reconnecting" || props.connectionStatus === "disconnected") {
    return props.connectionStatus;
  }

  if (resolvedConnectionState.value === "ready") {
    return "connected";
  }
  if (resolvedConnectionState.value === "loading") {
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
  return props.hubLabel ?? getCurrentWorkspace()?.hub_url ?? "local://workspace";
});

const identityLabel = computed(() => {
  if (props.runtimeMode) {
    return firstNonEmpty(
      (props.userLabel ?? "").trim(),
      (authStore.me?.display_name ?? "").trim(),
      (authStore.me?.user_id ?? "").trim(),
      "local-user"
    );
  }
  return props.roleLabel ?? authStore.me?.role ?? "Owner";
});

function firstNonEmpty(...values: string[]): string {
  for (const value of values) {
    if (value !== "") {
      return value;
    }
  }
  return "";
}
</script>

<style scoped>
.status-bar {
  height: 36px;
  border-radius: 0 0 var(--global-radius-12) var(--global-radius-12);
  background: transparent;
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
  border-radius: var(--global-radius-full);
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
