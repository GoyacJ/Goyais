<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="top">
      <WorkspaceSwitcherCard
        :workspaces="workspaces"
        :current-workspace-id="currentWorkspaceId"
        :collapsed="collapsed"
        :show-collapse-toggle="true"
        :can-create-workspace="true"
        :fallback-label="currentWorkspaceName"
        @switch-workspace="onSwitchWorkspace"
        @create-workspace="openCreateWorkspace"
        @toggle-collapse="collapsed = !collapsed"
      />

      <div class="projects-header">
        <span class="title">
          <AppIcon name="folder" :size="12" />
          <template v-if="!collapsed">项目</template>
        </span>
        <button class="icon-btn tiny" type="button" @click="pickDirectory">
          <AppIcon name="plus" :size="12" />
        </button>
      </div>

      <input
        ref="directoryInputRef"
        type="file"
        webkitdirectory
        directory
        multiple
        class="hidden-input"
        @change="onDirectoryPicked"
      />

      <div v-if="!collapsed" class="project-tree">
        <section v-for="project in projects" :key="project.id" class="project-node">
          <div class="project-row">
            <button class="tree-btn" type="button" @click="toggleProject(project.id)">
              <AppIcon :name="isProjectOpen(project.id) ? 'chevron-down' : 'chevron-right'" :size="12" />
              <AppIcon :name="isProjectOpen(project.id) ? 'folder-open' : 'folder'" :size="12" />
              <span>{{ project.name }}</span>
            </button>

            <div class="row-actions">
              <button class="icon-btn tiny" type="button" @click="$emit('addConversation', project.id)">
                <AppIcon name="plus" :size="10" />
              </button>
              <button class="icon-btn tiny" type="button" @click="$emit('deleteProject', project.id)">
                <AppIcon name="trash-2" :size="10" />
              </button>
            </div>
          </div>

          <div v-if="isProjectOpen(project.id)" class="conversation-list">
            <div
              v-for="conversation in conversationsByProjectId[project.id] ?? []"
              :key="conversation.id"
              class="conversation-item"
              :class="{ active: conversation.id === activeConversationId }"
            >
              <button
                class="conversation-main"
                type="button"
                @click="$emit('selectConversation', project.id, conversation.id)"
              >
                {{ conversation.name }}
              </button>
              <div class="row-actions">
                <button class="icon-btn tiny" type="button" @click.stop="$emit('exportConversation', conversation.id)">
                  <AppIcon name="file-down" :size="10" />
                </button>
                <button class="icon-btn tiny" type="button" @click.stop="$emit('deleteConversation', project.id, conversation.id)">
                  <AppIcon name="trash-2" :size="10" />
                </button>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>

    <UserProfileMenuCard
      class="bottom"
      :collapsed="collapsed"
      :avatar="userInitial"
      :title="currentWorkspaceMode === 'local' ? '本地' : userName"
      :subtitle="currentWorkspaceMode === 'local' ? 'Local Workspace' : 'Current Workspace'"
      :items="userMenuItems"
      @select="handleUserMenuSelect"
    />

    <div v-if="createWorkspaceOpen" class="modal-mask" @click.self="createWorkspaceOpen = false">
      <div class="modal">
        <h4>新增工作区</h4>
        <label>
          Hub 地址
          <input v-model="workspaceForm.hub_url" type="url" placeholder="https://hub.example.com" />
        </label>
        <label>
          用户名
          <input v-model="workspaceForm.username" type="text" placeholder="admin" />
        </label>
        <label>
          密码
          <input v-model="workspaceForm.password" type="password" placeholder="******" />
        </label>
        <div class="modal-actions">
          <button type="button" @click="createWorkspaceOpen = false">取消</button>
          <button type="button" @click="submitWorkspaceCreate">创建</button>
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";

import AppIcon from "@/shared/ui/AppIcon.vue";
import type { Conversation, Project, Workspace, WorkspaceMode } from "@/shared/types/api";
import UserProfileMenuCard from "@/shared/ui/sidebar/UserProfileMenuCard.vue";
import WorkspaceSwitcherCard from "@/shared/ui/sidebar/WorkspaceSwitcherCard.vue";

const props = defineProps<{
  workspaces: Workspace[];
  currentWorkspaceId: string;
  workspaceMode: WorkspaceMode;
  workspaceName: string;
  userName: string;
  projects: Project[];
  conversationsByProjectId: Record<string, Conversation[]>;
  activeConversationId: string;
}>();

const emit = defineEmits<{
  (event: "switchWorkspace", workspaceId: string): void;
  (event: "createWorkspace", payload: { hub_url: string; username: string; password: string }): void;
  (event: "importProject", repoPath: string): void;
  (event: "addConversation", projectId: string): void;
  (event: "deleteProject", projectId: string): void;
  (event: "exportConversation", conversationId: string): void;
  (event: "deleteConversation", projectId: string, conversationId: string): void;
  (event: "selectConversation", projectId: string, conversationId: string): void;
  (event: "openAccount"): void;
  (event: "openSettings"): void;
}>();

const collapsed = ref(false);
const createWorkspaceOpen = ref(false);
const directoryInputRef = ref<HTMLInputElement>();

const projectOpen = reactive<Record<string, boolean>>({});

const workspaceForm = reactive({
  hub_url: "",
  username: "",
  password: ""
});

const currentWorkspaceMode = computed(() => props.workspaceMode);
const currentWorkspaceName = computed(() => props.workspaceName);
const userInitial = computed(() => (props.userName || "L").slice(0, 1).toUpperCase());
const userMenuItems = computed(() => {
  const items = [
    {
      key: "settings",
      label: currentWorkspaceMode.value === "local" ? "本地设置" : "设置",
      icon: "settings"
    }
  ];
  if (currentWorkspaceMode.value !== "local") {
    items.unshift({ key: "account", label: "账号信息", icon: "circle-user-round" });
  }
  return items;
});

function toggleProject(projectId: string): void {
  projectOpen[projectId] = !isProjectOpen(projectId);
}

function isProjectOpen(projectId: string): boolean {
  return projectOpen[projectId] ?? true;
}

function pickDirectory(): void {
  directoryInputRef.value?.click();
}

function onDirectoryPicked(event: Event): void {
  const input = event.target as HTMLInputElement;
  const files = input.files;
  if (!files || files.length === 0) {
    return;
  }

  const first = files[0];
  const relative = (first as File & { webkitRelativePath?: string }).webkitRelativePath ?? "";
  const folder = relative.split("/")[0] || first.name;
  emit("importProject", `/imported/${folder}`);
  input.value = "";
}

function onSwitchWorkspace(workspaceId: string): void {
  emit("switchWorkspace", workspaceId);
}

function openCreateWorkspace(): void {
  createWorkspaceOpen.value = true;
}

function handleUserMenuSelect(key: string): void {
  if (key === "account") {
    emit("openAccount");
    return;
  }
  emit("openSettings");
}

function submitWorkspaceCreate(): void {
  if (workspaceForm.hub_url.trim() === "" || workspaceForm.username.trim() === "" || workspaceForm.password.trim() === "") {
    return;
  }

  emit("createWorkspace", {
    hub_url: workspaceForm.hub_url.trim(),
    username: workspaceForm.username.trim(),
    password: workspaceForm.password
  });

  workspaceForm.hub_url = "";
  workspaceForm.username = "";
  workspaceForm.password = "";
  createWorkspaceOpen.value = false;
}
</script>

<style scoped>
.sidebar {
  background: var(--semantic-surface);
  border-radius: var(--global-radius-12);
  padding: var(--global-space-12);
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto;
  gap: var(--global-space-12);
  transition: width 0.2s ease;
  width: 320px;
}

.sidebar.collapsed {
  width: 88px;
  padding: var(--global-space-8);
}

.top,
.project-tree {
  display: grid;
  gap: var(--global-space-8);
}

.top {
  min-height: 0;
  align-content: start;
  overflow: visible;
}

.icon-btn,
.tree-btn,
.conversation-item {
  border: 0;
  background: var(--semantic-surface);
  color: var(--semantic-text);
  border-radius: var(--global-radius-8);
  padding: 0;
}

.title {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
}

.projects-header,
.project-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.projects-header {
  margin-top: var(--global-space-12);
}

.title {
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
  font-weight: var(--global-font-weight-600);
}

.tiny {
  width: 20px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
}

.project-node {
  background: transparent;
  border-radius: var(--global-radius-8);
}

.tree-btn {
  background: transparent;
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
  padding: var(--global-space-4) 0;
}

.row-actions {
  display: inline-flex;
  gap: var(--global-space-4);
}

.conversation-list {
  margin-left: var(--component-tree-indent);
  border-left: 1px solid var(--semantic-divider);
  padding-left: var(--global-space-8);
  display: grid;
  gap: var(--global-space-4);
}

.project-tree {
  min-height: 0;
  align-content: start;
  overflow: auto;
}

.sidebar.collapsed .projects-header {
  margin-top: var(--global-space-8);
  display: grid;
  justify-items: center;
  gap: var(--global-space-8);
}

.sidebar.collapsed .title,
.sidebar.collapsed .tiny {
  justify-content: center;
  margin: 0 auto;
}

.conversation-item {
  background: transparent;
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  color: var(--semantic-text-muted);
  border-radius: var(--global-radius-6);
  gap: var(--global-space-8);
}

.conversation-main {
  border: 0;
  background: transparent;
  color: inherit;
  text-align: left;
  padding: var(--global-space-8);
}

.conversation-item.active {
  background: var(--component-sidebar-item-bg-active);
  color: var(--semantic-text);
}

.hidden-input {
  display: none;
}

.modal-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: grid;
  place-items: center;
  z-index: 20;
}

.modal {
  width: 360px;
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  border: 1px solid var(--semantic-border);
  padding: var(--global-space-12);
  display: grid;
  gap: var(--global-space-8);
}

.modal h4 {
  margin: 0;
  font-size: var(--global-font-size-14);
}

.modal label {
  display: grid;
  gap: var(--global-space-4);
  font-size: var(--global-font-size-11);
  color: var(--semantic-text-muted);
}

.modal input {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  color: var(--semantic-text);
  padding: var(--global-space-8);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--global-space-8);
}

.modal-actions button {
  border: 0;
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface-2);
  color: var(--semantic-text);
  padding: var(--global-space-8) var(--global-space-12);
}
</style>
