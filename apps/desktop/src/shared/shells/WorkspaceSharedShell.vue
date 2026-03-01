<template>
  <AccountShell
    v-if="shellKind === 'account'"
    :active-key="activeKey"
    :title="title"
    :subtitle="accountSubtitle"
    :scope-hint="scopeHint"
    :runtime-status-mode="runtimeStatus.runtimeStatusMode.value"
    :runtime-conversation-status="runtimeStatus.conversationStatus.value"
    :runtime-connection-status="runtimeStatus.connectionStatus.value"
    :runtime-user-display-name="runtimeStatus.userDisplayName.value"
    :runtime-hub-url="runtimeStatus.hubUrl.value"
  >
    <slot />
  </AccountShell>

  <SettingsShell
    v-else
    :active-key="activeKey"
    :title="title"
    :subtitle="settingsSubtitle"
    :runtime-status-mode="runtimeStatus.runtimeStatusMode.value"
    :runtime-conversation-status="runtimeStatus.conversationStatus.value"
    :runtime-connection-status="runtimeStatus.connectionStatus.value"
    :runtime-user-display-name="runtimeStatus.userDisplayName.value"
    :runtime-hub-url="runtimeStatus.hubUrl.value"
  >
    <slot />
  </SettingsShell>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { useConfigRuntimeStatus } from "@/shared/layouts/useConfigRuntimeStatus";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import AccountShell from "@/shared/shells/AccountShell.vue";
import SettingsShell from "@/shared/shells/SettingsShell.vue";
import { resolveWorkspaceSharedShell } from "@/shared/shells/workspaceSharedShell";

const props = withDefaults(
  defineProps<{
    activeKey: string;
    title: string;
    accountSubtitle: string;
    settingsSubtitle: string;
    scopeHint?: string;
  }>(),
  {
    scopeHint: "共享模块：可从账号信息或设置进入；根据当前工作区与权限显示不同能力。"
  }
);

const shellKind = computed(() => resolveWorkspaceSharedShell(workspaceStore.mode));
const runtimeStatus = useConfigRuntimeStatus();
</script>
