<template>
  <RemoteConfigLayout
    :active-key="activeKey"
    :menu-entries="resolvedMenuEntries"
    :scope-hint="scopeHint"
    :title="title"
    :subtitle="subtitle"
    :runtime-status-mode="runtimeStatusMode"
    :runtime-conversation-status="runtimeConversationStatus"
    :runtime-connection-status="runtimeConnectionStatus"
    :runtime-user-display-name="runtimeUserDisplayName"
    :runtime-hub-url="runtimeHubUrl"
  >
    <slot />
  </RemoteConfigLayout>
</template>

<script setup lang="ts">
import { computed } from "vue";

import type { MenuEntry } from "@/shared/navigation/pageMenus";
import { useRemoteConfigMenu } from "@/shared/navigation/pageMenus";
import type { ConnectionStatus, ConversationStatus } from "@/shared/types/api";
import RemoteConfigLayout from "@/shared/layouts/RemoteConfigLayout.vue";

const props = withDefaults(
  defineProps<{
    activeKey: string;
    title: string;
    subtitle: string;
    scopeHint?: string;
    menuEntries?: MenuEntry[];
    runtimeStatusMode?: boolean;
    runtimeConversationStatus?: ConversationStatus;
    runtimeConnectionStatus?: ConnectionStatus;
    runtimeUserDisplayName?: string;
    runtimeHubUrl?: string;
  }>(),
  {
    scopeHint: "Remote 视图：显示成员与角色、权限与审计，并按 RBAC+ABAC 控制",
    menuEntries: undefined,
    runtimeStatusMode: false,
    runtimeConversationStatus: "stopped",
    runtimeConnectionStatus: "disconnected",
    runtimeUserDisplayName: "",
    runtimeHubUrl: ""
  }
);

const menuEntries = useRemoteConfigMenu();
const resolvedMenuEntries = computed(() => props.menuEntries ?? menuEntries.value);
</script>
