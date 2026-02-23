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

      <HubStatusBar />
    </section>
  </div>
</template>

<script setup lang="ts">
import type { MenuEntry } from "@/shared/navigation/pageMenus";
import RemoteConfigSidebar from "@/shared/layouts/RemoteConfigSidebar.vue";
import HubStatusBar from "@/shared/ui/HubStatusBar.vue";
import StatusBadge from "@/shared/ui/StatusBadge.vue";
import Topbar from "@/shared/ui/Topbar.vue";

defineProps<{
  title: string;
  subtitle: string;
  scopeHint: string;
  activeKey: string;
  menuEntries: MenuEntry[];
}>();
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
  align-content: start;
}
</style>
