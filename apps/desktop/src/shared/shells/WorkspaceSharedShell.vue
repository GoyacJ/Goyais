<template>
  <AccountShell
    v-if="shellKind === 'account'"
    :active-key="activeKey"
    :title="title"
    :subtitle="accountSubtitle"
    :scope-hint="scopeHint"
  >
    <slot />
  </AccountShell>

  <SettingsShell
    v-else
    :active-key="activeKey"
    :title="title"
    :subtitle="settingsSubtitle"
  >
    <slot />
  </SettingsShell>
</template>

<script setup lang="ts">
import { computed } from "vue";

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
</script>
