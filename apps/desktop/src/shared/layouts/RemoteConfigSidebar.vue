<template>
  <aside class="sidebar">
    <div class="sidebar-top">
      <div class="mac-row">
        <span class="dot danger"></span>
        <span class="dot warning"></span>
        <span class="dot success"></span>
      </div>

      <button class="workspace-trigger" type="button" @click="workspaceMenuOpen = !workspaceMenuOpen">
        <span class="workspace-left">
          <IconSymbol name="work" :size="14" />
          <span>{{ workspaceLabel }}</span>
        </span>
        <IconSymbol name="expand_more" :size="14" />
      </button>

      <div v-if="workspaceMenuOpen" class="workspace-menu">
        <button
          v-for="workspace in workspaceStore.workspaces"
          :key="workspace.id"
          type="button"
          class="workspace-option"
          :class="{ active: workspace.id === workspaceStore.currentWorkspaceId }"
          @click="switchWorkspace(workspace.id)"
        >
          <IconSymbol :name="workspace.mode === 'local' ? 'home' : 'cloud'" :size="12" />
          <span>{{ workspace.name }}</span>
        </button>
      </div>

      <p class="scope-hint">{{ scopeHint }}</p>
      <p class="menu-title">工作区配置 Workspace Config</p>

      <nav class="menu-list">
        <RouterLink
          v-for="item in menuEntries"
          :key="item.key"
          :to="item.path"
          class="menu-item"
          :class="{ active: item.key === activeKey, muted: item.visibility !== 'enabled' }"
          @click.prevent="onMenuClick(item)"
        >
          {{ item.label }}
        </RouterLink>
      </nav>

      <p class="remote-hint">
        当前为 Remote：可见性=hidden/disabled/readonly/enabled，ABAC 拒绝返回 403
      </p>
    </div>

    <div class="sidebar-bottom">
      <button class="user-trigger" type="button" @click="userMenuOpen = !userMenuOpen">
        <span class="user-left">
          <span class="avatar">G</span>
          <span class="user-meta">
            <strong>{{ accountName }}</strong>
            <small>Owner · Current Workspace</small>
          </span>
        </span>
        <IconSymbol name="keyboard_arrow_up" :size="12" />
      </button>

      <div v-if="userMenuOpen" class="user-menu">
        <RouterLink class="menu-item small" to="/remote/account">账号信息 Profile</RouterLink>
        <RouterLink class="menu-item small" to="/settings/theme">设置 Settings</RouterLink>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";

import { switchWorkspaceContext } from "@/modules/workspace/store";
import type { MenuEntry } from "@/shared/navigation/pageMenus";
import { refreshNavigationVisibility } from "@/shared/stores/navigationStore";
import { authStore } from "@/shared/stores/authStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import IconSymbol from "@/shared/ui/IconSymbol.vue";

const props = defineProps<{
  activeKey: string;
  scopeHint: string;
  menuEntries: MenuEntry[];
}>();

const router = useRouter();
const workspaceMenuOpen = ref(false);
const userMenuOpen = ref(false);

const workspaceLabel = computed(() => {
  const current = workspaceStore.workspaces.find((item) => item.id === workspaceStore.currentWorkspaceId);
  return current ? current.name : "远程工作区 hub-prod";
});

const accountName = computed(() => authStore.me?.display_name ?? "local-user");

async function switchWorkspace(workspaceId: string): Promise<void> {
  await switchWorkspaceContext(workspaceId);
  refreshNavigationVisibility();
  workspaceMenuOpen.value = false;
}

function onMenuClick(item: MenuEntry): void {
  if (item.visibility === "enabled" || item.visibility === "readonly") {
    void router.push(item.path);
  }
}
</script>

<style scoped>
.sidebar {
  background: var(--semantic-surface);
  border-radius: var(--global-radius-12);
  padding: var(--global-space-12);
  display: grid;
  grid-template-rows: 1fr auto;
  gap: var(--global-space-12);
}
.sidebar-top,
.sidebar-bottom {
  display: grid;
  gap: var(--global-space-8);
  align-content: start;
}
.mac-row {
  display: inline-flex;
  gap: var(--global-space-8);
}
.dot {
  width: 12px;
  height: 12px;
  border-radius: 999px;
}
.danger { background: var(--semantic-danger); }
.warning { background: var(--semantic-warning); }
.success { background: var(--semantic-success); }
.workspace-trigger,
.user-trigger {
  border: 0;
  background: var(--semantic-surface-2);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8) var(--global-space-12);
  color: var(--semantic-text);
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.workspace-left,
.user-left {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
}
.workspace-menu,
.user-menu {
  background: var(--semantic-bg);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8);
  display: grid;
  gap: var(--global-space-4);
}
.workspace-option {
  border: 0;
  background: transparent;
  color: var(--semantic-text-muted);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8);
  display: flex;
  align-items: center;
  gap: var(--global-space-8);
}
.workspace-option.active,
.workspace-option:hover {
  background: var(--component-sidebar-item-bg-active);
  color: var(--semantic-text);
}
.scope-hint,
.menu-title,
.remote-hint {
  margin: 0;
  font-size: var(--global-font-size-11);
  color: var(--semantic-text-subtle);
}
.remote-hint {
  color: var(--component-toast-warning-fg);
  background: var(--component-toast-warning-bg);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8);
}
.menu-list {
  display: grid;
  gap: var(--global-space-4);
}
.menu-item {
  color: var(--semantic-text-muted);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8) var(--global-space-12);
}
.menu-item.active {
  color: var(--semantic-text);
  background: var(--component-sidebar-item-bg-active);
}
.menu-item.muted {
  opacity: var(--component-tree-item-disabled-opacity);
}
.menu-item.small {
  padding: var(--global-space-8);
}
.avatar {
  width: 24px;
  height: 24px;
  border-radius: 999px;
  background: var(--semantic-primary);
  color: var(--semantic-bg);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: var(--global-font-size-11);
  font-weight: var(--global-font-weight-700);
}
.user-meta {
  display: grid;
  gap: 2px;
  text-align: left;
}
.user-meta small {
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
}
</style>
