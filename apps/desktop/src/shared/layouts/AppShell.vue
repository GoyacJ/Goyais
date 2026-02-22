<template>
  <div class="app-shell">
    <aside class="shell-sidebar">
      <section class="sidebar-section">
        <h1 class="brand-title">Goyais v0.4.0</h1>
        <label class="workspace-label">
          <span>工作区 / Workspace</span>
          <select v-model="selectedWorkspaceId" @change="switchWorkspace" data-testid="workspace-switcher">
            <option v-for="workspace in workspaceOptions" :key="workspace.id" :value="workspace.id">
              {{ workspace.name }} · {{ workspace.mode }}
            </option>
          </select>
        </label>
      </section>

      <section class="sidebar-section">
        <h2 class="section-title">项目列表 / Projects</h2>
        <ul class="project-list">
          <li v-for="project in projectItems" :key="project.id">
            <button
              type="button"
              class="project-item"
              :class="{ active: activeProjectId === project.id }"
              @click="activeProjectId = project.id"
            >
              <span class="project-name">{{ project.name }}</span>
              <span class="project-meta">{{ project.meta }}</span>
            </button>
          </li>
        </ul>
      </section>

      <section class="sidebar-section">
        <h2 class="section-title">菜单 / Menu</h2>
        <nav class="menu-list">
          <RouterLink v-for="item in menuItems" :key="item.to" :to="item.to" class="menu-item">
            {{ item.label }}
          </RouterLink>
        </nav>
      </section>

      <section class="sidebar-bottom">
        <button type="button" class="settings-trigger" @click="settingsOpen = !settingsOpen" data-testid="settings-toggle">
          Settings
        </button>
        <div v-if="settingsOpen" class="settings-panel" data-testid="settings-panel">
          <h3>当前工作区账号信息</h3>
          <p>{{ currentWorkspace?.name ?? "未选择工作区" }}</p>
          <p>mode: {{ currentWorkspace?.mode ?? workspaceStore.mode }}</p>
          <p>user: {{ authStore.me?.display_name ?? "未登录 / local default" }}</p>
          <p>role: {{ authStore.me?.role ?? "admin(local)" }}</p>

          <h3>当前登录用户拥有的菜单</h3>
          <ul class="settings-list">
            <li v-for="item in menuItems" :key="`menu-${item.to}`">{{ item.label }}</li>
          </ul>

          <h3>设置</h3>
          <ul class="settings-list">
            <li>主题：Dark / Light</li>
            <li>语言：zh-CN / en-US</li>
            <li>诊断：Logs / Trace</li>
          </ul>
        </div>
      </section>
    </aside>

    <main class="shell-content">
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";

import { authStore, canAccessAdmin, refreshMeForCurrentWorkspace } from "@/shared/stores/authStore";
import { getCurrentWorkspace, setCurrentWorkspace, workspaceStore } from "@/shared/stores/workspaceStore";

const showAdminLink = computed(() => canAccessAdmin());
const settingsOpen = ref(false);
const activeProjectId = ref("project-main");

const currentWorkspace = computed(() => getCurrentWorkspace());

type WorkspaceOption = {
  id: string;
  name: string;
  mode: string;
};

const workspaceOptions = computed<WorkspaceOption[]>(() => {
  if (workspaceStore.workspaces.length > 0) {
    return workspaceStore.workspaces.map((workspace) => ({
      id: workspace.id,
      name: workspace.name,
      mode: workspace.mode
    }));
  }

  return [
    {
      id: workspaceStore.currentWorkspaceId || "ws_local_default",
      name: "Local Workspace",
      mode: workspaceStore.mode
    }
  ];
});

const selectedWorkspaceId = ref(workspaceOptions.value[0]?.id ?? "");

watch(
  () => workspaceStore.currentWorkspaceId,
  (workspaceId) => {
    if (workspaceId !== "") {
      selectedWorkspaceId.value = workspaceId;
    }
  }
);

const projectItems = computed(() => {
  const prefix = currentWorkspace.value?.name ?? "当前工作区";
  return [
    { id: "project-main", name: `${prefix} · 主项目`, meta: "最近活跃" },
    { id: "project-exec", name: "执行调度 / Execution", meta: "3 Conversations" },
    { id: "project-perm", name: "权限治理 / Permission", meta: "RBAC + ABAC" }
  ];
});

const menuItems = computed(() => {
  const base = [
    { to: "/workspace", label: "Workspace" },
    { to: "/project", label: "Project" },
    { to: "/conversation", label: "Conversation" },
    { to: "/resource", label: "Resource" }
  ];

  if (showAdminLink.value) {
    base.push({ to: "/admin", label: "Admin" });
  }

  return base;
});

async function switchWorkspace(): Promise<void> {
  if (selectedWorkspaceId.value === "") {
    return;
  }

  setCurrentWorkspace(selectedWorkspaceId.value);

  try {
    await refreshMeForCurrentWorkspace();
  } catch {
    // UI shell keeps interaction responsive; error state is handled in stores.
  }
}
</script>

<style scoped>
.app-shell {
  height: 100vh;
  display: grid;
  grid-template-columns: 280px 1fr;
  background: var(--component-shell-content-bg);
}

.shell-sidebar {
  display: grid;
  grid-template-rows: auto auto auto 1fr;
  gap: var(--component-space-md);
  padding: var(--component-space-md);
  background: linear-gradient(180deg, var(--component-shell-sidebar-bg), var(--component-shell-sidebar-bg-muted));
  border-right: 1px solid var(--component-shell-divider);
}

.sidebar-section {
  display: grid;
  gap: var(--component-space-sm);
}

.brand-title {
  margin: 0;
  font-size: var(--component-font-size-title);
  font-weight: var(--global-font-weight-700);
  color: var(--semantic-text-primary);
}

.workspace-label {
  display: grid;
  gap: var(--component-space-xs);
  color: var(--component-text-subtle);
  font-size: var(--global-font-size-12);
}

select {
  width: 100%;
  border: 1px solid var(--component-panel-border);
  border-radius: var(--component-radius-sm);
  background: var(--component-shell-content-bg-elevated);
  color: var(--component-text-main);
  padding: 8px 10px;
  font: inherit;
}

.section-title {
  margin: 0;
  font-size: var(--global-font-size-12);
  font-weight: var(--global-font-weight-600);
  color: var(--component-text-subtle);
}

.project-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: var(--component-space-xs);
}

.project-item {
  width: 100%;
  border: 1px solid transparent;
  border-radius: var(--component-radius-sm);
  background: transparent;
  color: var(--component-text-main);
  padding: 8px 10px;
  text-align: left;
  display: grid;
  gap: 2px;
  cursor: pointer;
}

.project-item:hover {
  background: var(--component-sidebar-item-bg-hover);
}

.project-item.active {
  border-color: var(--component-shell-divider);
  background: var(--component-sidebar-item-bg-active);
}

.project-name {
  font-size: var(--global-font-size-13);
}

.project-meta {
  color: var(--component-text-subtle);
  font-size: var(--global-font-size-11);
}

.menu-list {
  display: grid;
  gap: 4px;
}

.menu-item {
  border-radius: var(--component-radius-sm);
  padding: 8px 10px;
  color: var(--component-text-subtle);
}

.menu-item:hover {
  background: var(--component-sidebar-item-bg-hover);
  color: var(--component-text-main);
}

.sidebar-bottom {
  align-self: end;
  display: grid;
  gap: var(--component-space-sm);
}

.settings-trigger {
  border: 1px solid var(--component-shell-divider);
  border-radius: var(--component-radius-sm);
  background: transparent;
  color: var(--component-text-main);
  padding: 8px 10px;
  text-align: left;
  cursor: pointer;
}

.settings-trigger:hover {
  background: var(--component-sidebar-item-bg-hover);
}

.settings-panel {
  border: 1px solid var(--component-shell-settings-border);
  border-radius: var(--component-radius-md);
  background: var(--component-shell-settings-bg);
  padding: var(--component-space-sm);
  display: grid;
  gap: 6px;
  color: var(--component-text-main);
}

.settings-panel h3 {
  margin: 8px 0 0;
  font-size: var(--global-font-size-12);
  font-weight: var(--global-font-weight-600);
  color: var(--component-text-subtle);
}

.settings-panel p {
  margin: 0;
  font-size: var(--global-font-size-12);
}

.settings-list {
  margin: 0;
  padding-left: 16px;
  display: grid;
  gap: 2px;
}

.settings-list li {
  font-size: var(--global-font-size-12);
  color: var(--component-text-main);
}

.shell-content {
  min-width: 0;
  overflow: auto;
  padding: var(--component-space-md);
  background: var(--component-shell-content-bg);
}

@media (max-width: 980px) {
  .app-shell {
    grid-template-columns: 1fr;
  }

  .shell-sidebar {
    grid-template-rows: auto;
  }

  .shell-content {
    padding-top: 0;
  }
}

button,
a {
  font: inherit;
}

/* Keep contrast alignment with dark two-tone shell */
.shell-content :deep(section),
.shell-content :deep(.panel),
.shell-content :deep(.events),
.shell-content :deep(.composer),
.shell-content :deep(.sidebar) {
  background: var(--component-shell-content-bg-elevated);
}

.shell-content :deep(.notice),
.shell-content :deep(.status) {
  color: var(--component-text-subtle);
}

.shell-content :deep(.error) {
  color: var(--danger);
}

.shell-content :deep(button) {
  border: 1px solid var(--component-panel-border);
  border-radius: var(--component-radius-sm);
  background: var(--component-shell-content-bg-elevated);
  color: var(--component-text-main);
  padding: 8px 10px;
}

.shell-content :deep(input),
.shell-content :deep(textarea) {
  border: 1px solid var(--component-panel-border);
  border-radius: var(--component-radius-sm);
  background: var(--component-shell-content-bg-elevated);
  color: var(--component-text-main);
  padding: 8px 10px;
}

.shell-content :deep(.workspace-row) {
  display: flex;
  align-items: center;
  gap: var(--component-space-sm);
}
</style>
