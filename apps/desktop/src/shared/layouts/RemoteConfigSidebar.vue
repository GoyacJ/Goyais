<template>
  <aside class="sidebar">
    <div class="sidebar-top">
      <WorkspaceSwitcherCard
        :workspaces="workspaceStore.workspaces"
        :current-workspace-id="workspaceStore.currentWorkspaceId"
        fallback-label="工作区"
        @switch-workspace="switchWorkspace"
      />

      <button class="menu-item nav-main" type="button" @click="goMain">
        <AppIcon name="house" :size="12" />
        <span>主界面</span>
      </button>

      <p class="group-title">远程管理</p>
      <nav class="menu-list">
        <RouterLink
          v-for="item in remoteEntries"
          :key="item.key"
          :to="item.path"
          class="menu-item"
          :class="{ active: item.key === activeKey, muted: item.visibility !== 'enabled' }"
          :data-visibility="item.visibility"
          @click.prevent="onMenuClick(item)"
        >
          <AppIcon :name="resolveMenuIcon(item.key)" :size="12" />
          <span>{{ item.label }}</span>
          <small v-if="item.visibility === 'readonly'" class="visibility-tag">只读</small>
          <small v-else-if="item.visibility === 'disabled'" class="visibility-tag">禁用</small>
        </RouterLink>
      </nav>

      <p class="group-title">工作区配置</p>
      <nav class="menu-list">
        <RouterLink
          v-for="item in workspaceEntries"
          :key="item.key"
          :to="item.path"
          class="menu-item"
          :class="{ active: item.key === activeKey, muted: item.visibility !== 'enabled' }"
          :data-visibility="item.visibility"
          @click.prevent="onMenuClick(item)"
        >
          <AppIcon :name="resolveMenuIcon(item.key)" :size="12" />
          <span>{{ item.label }}</span>
          <small v-if="item.visibility === 'readonly'" class="visibility-tag">只读</small>
          <small v-else-if="item.visibility === 'disabled'" class="visibility-tag">禁用</small>
        </RouterLink>
      </nav>
    </div>

    <UserProfileMenuCard
      class="sidebar-bottom"
      :avatar="accountInitial"
      :title="accountName"
      :subtitle="`${authStore.me?.role ?? 'owner'} · Current Workspace`"
      :items="userMenuItems"
      @select="onUserMenuSelect"
    />
  </aside>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRouter } from "vue-router";

import { switchWorkspaceContext } from "@/modules/workspace/store";
import type { MenuEntry } from "@/shared/navigation/pageMenus";
import { authStore } from "@/shared/stores/authStore";
import { refreshNavigationVisibility } from "@/shared/stores/navigationStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import AppIcon from "@/shared/ui/AppIcon.vue";
import UserProfileMenuCard from "@/shared/ui/sidebar/UserProfileMenuCard.vue";
import WorkspaceSwitcherCard from "@/shared/ui/sidebar/WorkspaceSwitcherCard.vue";

const props = defineProps<{
  activeKey: string;
  scopeHint: string;
  menuEntries: MenuEntry[];
}>();

const router = useRouter();
const remoteKeys = ["remote_account", "remote_members_roles", "remote_permissions_audit"];
const workspaceKeys = [
  "workspace_project_config",
  "workspace_agent",
  "workspace_model",
  "workspace_rules",
  "workspace_skills",
  "workspace_mcp"
];
const remoteEntries = computed(() =>
  props.menuEntries.filter((item) => remoteKeys.includes(item.key) && item.visibility !== "hidden")
);
const workspaceEntries = computed(() =>
  props.menuEntries.filter((item) => workspaceKeys.includes(item.key) && item.visibility !== "hidden")
);

const accountName = computed(() => authStore.me?.display_name ?? "local-user");
const accountInitial = computed(() => accountName.value.slice(0, 1).toUpperCase());
const userMenuItems = computed(() => [
  { key: "account", label: "账号信息", icon: "circle-user-round" },
  { key: "settings", label: "设置", icon: "settings" }
]);

async function switchWorkspace(workspaceId: string): Promise<void> {
  await switchWorkspaceContext(workspaceId);
  refreshNavigationVisibility();

  if (workspaceStore.mode === "local") {
    void router.push("/settings/theme");
    return;
  }

  if (!router.currentRoute.value.path.startsWith("/remote/")) {
    void router.push("/remote/account");
  }
}

function onMenuClick(item: MenuEntry): void {
  if (item.visibility === "enabled" || item.visibility === "readonly") {
    void router.push(item.path);
  }
}

function onUserMenuSelect(key: string): void {
  if (key === "account") {
    void router.push("/remote/account");
    return;
  }
  void router.push("/settings/theme");
}

function goMain(): void {
  void router.push("/main");
}

function resolveMenuIcon(key: string): string {
  if (key === "remote_account") {
    return "user-round";
  }
  if (key === "remote_members_roles") {
    return "users";
  }
  if (key === "remote_permissions_audit") {
    return "shield-check";
  }
  if (key === "workspace_project_config") {
    return "file-text";
  }
  if (key === "workspace_agent") {
    return "bot";
  }
  if (key === "workspace_model") {
    return "cpu";
  }
  if (key === "workspace_rules") {
    return "scroll-text";
  }
  if (key === "workspace_skills") {
    return "wrench";
  }
  return "plug-zap";
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

.group-title {
  margin: 0;
  font-size: var(--global-font-size-11);
  color: var(--semantic-text-subtle);
  font-weight: var(--global-font-weight-600);
}

.menu-list {
  display: grid;
  gap: var(--global-space-4);
}

.menu-item {
  color: var(--semantic-text-muted);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8) var(--global-space-12);
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
  justify-content: flex-start;
  transition: background 0.15s ease, color 0.15s ease;
  border: 0;
  font-size: var(--global-font-size-12);
}

.nav-main {
  background: transparent;
  text-align: left;
}

.menu-item:hover {
  background: var(--semantic-surface-2);
  color: var(--semantic-text);
}

.menu-item.active {
  color: var(--semantic-text);
  background: var(--component-sidebar-item-bg-active);
}

.menu-item.muted {
  opacity: var(--component-tree-item-disabled-opacity);
}

.visibility-tag {
  margin-left: auto;
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
}
</style>
