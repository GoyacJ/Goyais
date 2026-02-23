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
          <div class="chips">
            <StatusBadge tone="running" label="scope: local_workspace" />
            <span class="mode-tag">Local</span>
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
import LocalSettingsSidebar from "@/shared/layouts/LocalSettingsSidebar.vue";
import HubStatusBar from "@/shared/ui/HubStatusBar.vue";
import StatusBadge from "@/shared/ui/StatusBadge.vue";
import Topbar from "@/shared/ui/Topbar.vue";

defineProps<{
  title: string;
  subtitle: string;
  activeKey: string;
  menuEntries: MenuEntry[];
}>();
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
  align-items: center;
  gap: var(--global-space-8);
}

.mode-tag {
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-11);
}

.main {
  min-height: 0;
  overflow: auto;
  display: grid;
  gap: var(--global-space-12);
}
</style>
