<template>
  <div class="layout">
    <LocalSettingsSidebar :active-key="activeKey" :menu-entries="menuEntries" />

    <section class="content">
      <Topbar>
        <template #left>
          <div class="header-left">
            <strong>{{ title }}</strong>
            <span>{{ subtitle }}</span>
          </div>
        </template>
        <template #right>
          <div v-if="runtimeStatusMode" class="chips">
            <StatusBadge :tone="conversationStatusTone" :label="`conversation: ${runtimeConversationStatus}`" />
          </div>
          <div v-else class="chips">
            <StatusBadge tone="running" label="scope: local_workspace" />
            <span class="mode-tag">Local</span>
          </div>
        </template>
      </Topbar>

      <main class="main">
        <slot />
      </main>

      <HubStatusBar
        :runtime-mode="runtimeStatusMode"
        :hub-label="runtimeHubUrl"
        :user-label="runtimeUserDisplayName"
        :connection-status="runtimeConnectionStatus"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

import type { MenuEntry } from "@/shared/navigation/pageMenus";
import LocalSettingsSidebar from "@/shared/layouts/LocalSettingsSidebar.vue";
import type { ConnectionStatus, ConversationStatus } from "@/shared/types/api";
import HubStatusBar from "@/shared/ui/HubStatusBar.vue";
import StatusBadge from "@/shared/ui/StatusBadge.vue";
import Topbar from "@/shared/ui/Topbar.vue";

const props = withDefaults(
  defineProps<{
  title: string;
  subtitle: string;
  activeKey: string;
  menuEntries: MenuEntry[];
    runtimeStatusMode?: boolean;
    runtimeConversationStatus?: ConversationStatus;
    runtimeConnectionStatus?: ConnectionStatus;
    runtimeUserDisplayName?: string;
    runtimeHubUrl?: string;
  }>(),
  {
    runtimeStatusMode: false,
    runtimeConversationStatus: "stopped",
    runtimeConnectionStatus: "disconnected",
    runtimeUserDisplayName: "",
    runtimeHubUrl: ""
  }
);

const conversationStatusTone = computed(() => {
  switch (props.runtimeConversationStatus) {
    case "running":
      return "running";
    case "queued":
      return "queued";
    case "done":
      return "success";
    case "error":
      return "failed";
    default:
      return "cancelled";
  }
});
</script>

<style scoped>
.layout {
  height: 100vh;
  display: grid;
  grid-template-columns: 320px 1fr;
  gap: var(--global-space-8);
  padding: 0;
  background: var(--component-shell-bg);
}

.content {
  padding: 0 var(--global-space-8) 0 0;
  display: grid;
  grid-template-rows: auto 1fr auto;
  gap: var(--global-space-12);
  min-height: 0;
}

.header-left {
  display: inline-flex;
  gap: var(--global-space-8);
  align-items: center;
}

.header-left span {
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-12);
}

.chips {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
}

.mode-tag {
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-11);
}

.main {
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  display: grid;
  gap: var(--global-space-12);
  align-content: start;
}
</style>
