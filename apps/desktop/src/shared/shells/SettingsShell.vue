<template>
  <LocalSettingsLayout
    :active-key="activeKey"
    :menu-entries="resolvedMenuEntries"
    :title="title"
    :subtitle="subtitle"
    :runtime-status-mode="runtimeStatusMode"
    :runtime-conversation-status="runtimeConversationStatus"
    :runtime-connection-status="runtimeConnectionStatus"
    :runtime-user-display-name="runtimeUserDisplayName"
    :runtime-hub-url="runtimeHubUrl"
  >
    <slot />
  </LocalSettingsLayout>
</template>

<script setup lang="ts">
import { computed } from "vue";

import LocalSettingsLayout from "@/shared/layouts/LocalSettingsLayout.vue";
import type { MenuEntry } from "@/shared/navigation/pageMenus";
import { useLocalSettingsMenu } from "@/shared/navigation/pageMenus";
import type { ConnectionStatus, ConversationStatus } from "@/shared/types/api";

const props = withDefaults(
  defineProps<{
  activeKey: string;
  title: string;
  subtitle: string;
  menuEntries?: MenuEntry[];
    runtimeStatusMode?: boolean;
    runtimeConversationStatus?: ConversationStatus;
    runtimeConnectionStatus?: ConnectionStatus;
    runtimeUserDisplayName?: string;
    runtimeHubUrl?: string;
  }>(),
  {
    menuEntries: undefined,
    runtimeStatusMode: false,
    runtimeConversationStatus: "stopped",
    runtimeConnectionStatus: "disconnected",
    runtimeUserDisplayName: "",
    runtimeHubUrl: ""
  }
);

const menuEntries = useLocalSettingsMenu();
const resolvedMenuEntries = computed(() => props.menuEntries ?? menuEntries.value);
</script>
