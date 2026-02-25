<template>
  <div class="layout" :class="{ 'sidebar-open': sidebarOpen }">
    <button
      class="mobile-backdrop"
      type="button"
      aria-label="关闭导航菜单"
      @click="sidebarOpen = false"
    ></button>

    <aside class="sidebar-slot" @click="onSidebarClick">
      <div class="sidebar-slot-fill">
        <LocalSettingsSidebar :active-key="activeKey" :menu-entries="menuEntries" />
      </div>
    </aside>

    <section class="content">
      <button
        class="mobile-menu-button"
        type="button"
        aria-label="打开导航菜单"
        @click="sidebarOpen = true"
      >
        ≡
      </button>
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
import { computed, ref } from "vue";

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

const sidebarOpen = ref(false);

function onSidebarClick(event: MouseEvent): void {
  const target = event.target as HTMLElement | null;
  if (!target) {
    return;
  }
  if (target.closest("a,[role='link']")) {
    sidebarOpen.value = false;
  }
}
</script>

<style scoped>
.layout {
  height: 100dvh;
  display: grid;
  grid-template-columns: 320px 1fr;
  gap: var(--global-space-8);
  padding: 0;
  background: var(--component-shell-bg);
}

.sidebar-slot {
  min-height: 0;
  height: 100%;
}

.sidebar-slot-fill {
  height: 100%;
  display: grid;
}

.content {
  padding: 0 var(--global-space-8) 0 0;
  display: grid;
  grid-template-rows: auto 1fr auto;
  gap: var(--global-space-12);
  min-height: 0;
  position: relative;
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

.mobile-backdrop,
.mobile-menu-button {
  display: none;
}

@media (max-width: 768px) {
  .layout {
    grid-template-columns: minmax(0, 1fr);
    gap: 0;
    padding-top: var(--safe-area-top);
    padding-right: var(--safe-area-right);
    padding-bottom: var(--safe-area-bottom);
    padding-left: var(--safe-area-left);
  }

  .sidebar-slot {
    position: fixed;
    top: var(--safe-area-top);
    left: var(--safe-area-left);
    bottom: var(--safe-area-bottom);
    width: min(86vw, 340px);
    z-index: 40;
    transform: translateX(calc(-100% - var(--global-space-12)));
    transition: transform 0.2s ease;
  }

  .layout.sidebar-open .sidebar-slot {
    transform: translateX(0);
  }

  .content {
    padding: 0 var(--global-space-8);
    gap: var(--global-space-8);
  }

  .header-left {
    padding-left: 52px;
  }

  .mobile-menu-button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    position: absolute;
    top: var(--global-space-8);
    left: var(--global-space-8);
    width: 44px;
    height: 44px;
    border: 1px solid var(--semantic-border);
    border-radius: var(--global-radius-12);
    background: var(--semantic-surface);
    color: var(--semantic-text);
    z-index: 12;
  }

  .mobile-backdrop {
    display: block;
    position: fixed;
    inset: 0;
    border: 0;
    background: transparent;
    pointer-events: none;
    z-index: 32;
  }

  .layout.sidebar-open .mobile-backdrop {
    background: #00000066;
    pointer-events: auto;
  }
}
</style>
