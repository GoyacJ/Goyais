<template>
  <footer
    class="flex h-[36px] items-center justify-between gap-[var(--global-space-12)] rounded-b-[var(--global-radius-12)] bg-transparent px-[var(--global-space-12)] text-[var(--global-font-size-11)] text-[var(--semantic-text-muted)]"
  >
    <span class="inline-flex items-center gap-[var(--global-space-8)]">
      <AppIcon name="plug-zap" :size="12" />
      Hub: {{ hubLabel }}
    </span>
    <span class="inline-flex items-center gap-[var(--global-space-8)]">
      <span>{{ identityLabel }}</span>
      <span class="h-[8px] w-[8px] rounded-[var(--global-radius-full)] bg-[var(--semantic-text-subtle)]" :class="connectionClass"></span>
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
    return "text-[var(--semantic-success)]";
  }
  if (connectionLabel.value === "reconnecting") {
    return "text-[var(--semantic-warning)]";
  }
  return "text-[var(--semantic-danger)]";
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
