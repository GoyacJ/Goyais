<template>
  <div class="layout">
    <RemoteConfigSidebar :active-key="activeKey" :scope-hint="scopeHint" :menu-entries="menuEntries" />

    <section class="content">
      <Topbar>
        <template #left>
          <div class="header-left">
            <strong>{{ title }}</strong>
            <span>{{ subtitle }}</span>
          </div>
        </template>
        <template #right>
          <div class="chips">
            <StatusBadge tone="running" label="scope: current_workspace" />
            <StatusBadge tone="connected" label="Remote" />
          </div>
        </template>
      </Topbar>

      <main class="main">
        <slot />
      </main>

      <footer class="status-bar">
        <span>Hub: {{ hubLabel }}</span>
        <span>remote workspace · connected</span>
      </footer>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

import type { MenuEntry } from "@/shared/navigation/pageMenus";
import RemoteConfigSidebar from "@/shared/layouts/RemoteConfigSidebar.vue";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import StatusBadge from "@/shared/ui/StatusBadge.vue";
import Topbar from "@/shared/ui/Topbar.vue";

defineProps<{
  title: string;
  subtitle: string;
  scopeHint: string;
  activeKey: string;
  menuEntries: MenuEntry[];
}>();

const hubLabel = computed(() => {
  const current = workspaceStore.workspaces.find((item) => item.id === workspaceStore.currentWorkspaceId);
  return current?.hub_url ?? "https://hub-prod.goyais.io";
});
</script>

<style scoped>
.layout {
  height: 100vh;
  display: grid;
  grid-template-columns: 320px 1fr;
  gap: var(--global-space-12);
  padding: var(--global-space-12);
  background: var(--semantic-bg);
}
.content {
  padding: var(--global-space-8) var(--global-space-12) 0;
  display: grid;
  grid-template-rows: auto 1fr auto;
  gap: var(--global-space-12);
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
  gap: var(--global-space-8);
}
.main {
  min-height: 0;
  overflow: auto;
  display: grid;
  gap: var(--global-space-12);
}
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
}
</style>
