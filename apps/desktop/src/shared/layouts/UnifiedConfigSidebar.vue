<template>
  <aside class="sidebar">
    <div class="sidebar-top">
      <WorkspaceSwitcherCard
        :workspaces="workspaceStore.workspaces"
        :current-workspace-id="workspaceStore.currentWorkspaceId"
        :can-create-workspace="true"
        fallback-label="工作区"
        @switch-workspace="switchWorkspace"
        @create-workspace="openCreateWorkspace"
      />
      <button class="menu-item nav-main" type="button" @click="goMain">
        <AppIcon name="house" :size="12" />
        <span>主界面</span>
      </button>
      <template v-for="section in sections" :key="section.title">
        <p class="group-title">{{ section.title }}</p>
        <nav class="menu-list">
          <RouterLink
            v-for="item in resolveEntries(section.keys)"
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
      </template>
    </div>

    <UserProfileMenuCard
      class="sidebar-bottom"
      :avatar="profile.avatar"
      :title="profile.title"
      :subtitle="profile.subtitle"
      :items="profile.items"
      @select="onUserMenuSelect"
    />

    <WorkspaceCreateModal :open="createWorkspaceOpen" @close="createWorkspaceOpen = false" @submit="submitWorkspaceCreate" />
  </aside>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";

import { createRemoteConnection } from "@/modules/workspace/services";
import { setWorkspaceConnection, switchWorkspaceContext, upsertWorkspace } from "@/modules/workspace/store";
import type { MenuEntry } from "@/shared/navigation/pageMenus";
import { authStore, setWorkspaceToken } from "@/shared/stores/authStore";
import { refreshNavigationVisibility } from "@/shared/stores/navigationStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import AppIcon from "@/shared/ui/AppIcon.vue";
import UserProfileMenuCard from "@/shared/ui/sidebar/UserProfileMenuCard.vue";
import WorkspaceCreateModal from "@/shared/ui/sidebar/WorkspaceCreateModal.vue";
import WorkspaceSwitcherCard from "@/shared/ui/sidebar/WorkspaceSwitcherCard.vue";

type SidebarVariant = "local" | "remote";

type SidebarSection = {
  title: string;
  keys: string[];
};

const props = defineProps<{
  variant: SidebarVariant;
  activeKey: string;
  menuEntries: MenuEntry[];
}>();

const router = useRouter();
const createWorkspaceOpen = ref(false);

const sections = computed<SidebarSection[]>(() => {
  if (props.variant === "remote") {
    return [
      {
        title: "远程管理",
        keys: ["remote_account", "remote_members_roles", "remote_permissions_audit"]
      },
      {
        title: "工作区配置",
        keys: [
          "workspace_project_config",
          "workspace_agent",
          "workspace_model",
          "workspace_rules",
          "workspace_skills",
          "workspace_mcp"
        ]
      }
    ];
  }

  return [
    {
      title: "工作区配置",
      keys: [
        "workspace_project_config",
        "workspace_agent",
        "workspace_model",
        "workspace_rules",
        "workspace_skills",
        "workspace_mcp"
      ]
    },
    {
      title: "软件通用设置",
      keys: ["settings_theme", "settings_i18n", "settings_general"]
    }
  ];
});

const profile = computed(() => {
  if (props.variant === "local") {
    return {
      avatar: "L",
      title: "本地",
      subtitle: "Local Workspace",
      items: [{ key: "settings", label: "设置", icon: "settings" }]
    };
  }

  const displayName = authStore.me?.display_name ?? "local-user";
  return {
    avatar: displayName.slice(0, 1).toUpperCase(),
    title: displayName,
    subtitle: `${authStore.me?.role ?? "owner"} · Current Workspace`,
    items: [
      { key: "account", label: "账号信息", icon: "circle-user-round" },
      { key: "settings", label: "设置", icon: "settings" }
    ]
  };
});

function resolveEntries(keys: string[]): MenuEntry[] {
  return props.menuEntries.filter((item) => keys.includes(item.key) && item.visibility !== "hidden");
}

async function switchWorkspace(workspaceId: string): Promise<void> {
  await switchWorkspaceContext(workspaceId);
  refreshNavigationVisibility();

  if (workspaceStore.mode === "remote") {
    if (!router.currentRoute.value.path.startsWith("/remote/")) {
      void router.push("/remote/account");
    }
    return;
  }

  if (!router.currentRoute.value.path.startsWith("/settings/")) {
    void router.push("/settings/theme");
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

function openCreateWorkspace(): void {
  createWorkspaceOpen.value = true;
}

async function submitWorkspaceCreate(payload: { hub_url: string; username: string; password: string }): Promise<void> {
  const result = await createRemoteConnection(payload);
  upsertWorkspace(result.workspace);
  setWorkspaceConnection(result.connection);
  if (result.access_token) {
    setWorkspaceToken(result.workspace.id, result.access_token);
  }
  createWorkspaceOpen.value = false;
  await switchWorkspace(result.workspace.id);
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
  if (key === "workspace_mcp") {
    return "plug-zap";
  }
  if (key === "settings_i18n") {
    return "file-text";
  }
  return "settings";
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
