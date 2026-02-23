<template>
  <aside class="sidebar">
    <div class="top">
      <div class="mac-row">
        <span class="dot danger"></span>
        <span class="dot warning"></span>
        <span class="dot success"></span>
      </div>

      <div class="workspace-row">
        <button class="workspace-btn" type="button" @click="workspaceOpen = !workspaceOpen">
          <span class="workspace-label">
            <IconSymbol name="work" :size="13" />
            {{ workspaceName }}
          </span>
          <IconSymbol name="expand_more" :size="14" />
        </button>
        <button class="icon-btn" type="button" title="Collapse">
          <IconSymbol name="left_panel_close" :size="14" />
        </button>
      </div>

      <div v-if="workspaceOpen" class="workspace-list">
        <div class="workspace-item active">
          <IconSymbol name="radio_button_checked" :size="10" />
          <span>本地工作区 Local</span>
        </div>
        <div class="workspace-item">
          <IconSymbol name="radio_button_unchecked" :size="10" />
          <span>远程工作区 Remote</span>
        </div>
        <div class="workspace-item">
          <IconSymbol name="add" :size="12" />
          <span>新增工作区</span>
        </div>
      </div>

      <div class="projects-header">
        <span class="title">
          <IconSymbol name="folder" :size="12" />
          项目
        </span>
        <button class="icon-btn tiny" type="button" @click="$emit('addProject')">
          <IconSymbol name="add" :size="12" />
        </button>
      </div>

      <div class="project-tree">
        <section
          v-for="project in projects"
          :key="project.id"
          class="project-node"
        >
          <div class="project-row">
            <button class="tree-btn" type="button" @click="toggleProject(project.id)">
              <IconSymbol :name="isProjectOpen(project.id) ? 'keyboard_arrow_down' : 'keyboard_arrow_right'" :size="12" />
              <IconSymbol :name="isProjectOpen(project.id) ? 'folder_open' : 'folder'" :size="12" />
              <span>{{ project.name }}</span>
            </button>

            <div class="row-actions">
              <button class="icon-btn tiny" type="button" @click="$emit('addConversation', project.id)">
                <IconSymbol name="add" :size="12" />
              </button>
              <button class="icon-btn tiny" type="button" @click="$emit('deleteProject', project.id)">
                <IconSymbol name="remove" :size="12" />
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
              <button class="icon-btn tiny" type="button" @click.stop="$emit('deleteConversation', project.id, conversation.id)">
                <IconSymbol name="remove" :size="10" />
              </button>
            </div>
          </div>
        </section>
      </div>
    </div>

    <div class="bottom">
      <div class="row">
        <IconSymbol name="home" :size="12" />
        <span>当前：{{ workspaceMode === "local" ? "本地工作区 Local" : "远程工作区 Remote" }}</span>
      </div>
      <div class="row">
        <IconSymbol name="settings" :size="12" />
        <RouterLink to="/settings/theme">设置入口 -> 本地-主题</RouterLink>
      </div>
      <div class="row muted">
        <IconSymbol name="person" :size="12" />
        <span>账号信息入口 -> 仅远程工作区显示</span>
      </div>
      <p class="hint">Agent/模型/Rules/Skills/MCP 为共享页面；按入口与权限显示差异。</p>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";

import IconSymbol from "@/shared/ui/IconSymbol.vue";
import type { Conversation, Project, WorkspaceMode } from "@/shared/types/api";

defineProps<{
  workspaceName: string;
  workspaceMode: WorkspaceMode;
  projects: Project[];
  conversationsByProjectId: Record<string, Conversation[]>;
  activeConversationId: string;
}>();

defineEmits<{
  (event: "addProject"): void;
  (event: "deleteProject", projectId: string): void;
  (event: "addConversation", projectId: string): void;
  (event: "deleteConversation", projectId: string, conversationId: string): void;
  (event: "selectConversation", projectId: string, conversationId: string): void;
}>();

const workspaceOpen = ref(true);
const projectOpen = reactive<Record<string, boolean>>({});

function toggleProject(projectId: string): void {
  projectOpen[projectId] = !isProjectOpen(projectId);
}

function isProjectOpen(projectId: string): boolean {
  return projectOpen[projectId] ?? true;
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
.top,
.workspace-list,
.project-tree,
.bottom {
  display: grid;
  gap: var(--global-space-8);
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
.workspace-row {
  display: grid;
  grid-template-columns: 1fr 30px;
  gap: var(--global-space-8);
}
.workspace-btn,
.icon-btn,
.tree-btn,
.conversation-item {
  border: 0;
  background: var(--semantic-surface-2);
  color: var(--semantic-text);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8);
}
.workspace-btn {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.workspace-label,
.title {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
}
.workspace-item {
  background: var(--semantic-bg);
  color: var(--semantic-text-muted);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8);
  display: flex;
  align-items: center;
  gap: var(--global-space-8);
}
.workspace-item.active {
  background: var(--component-sidebar-item-bg-active);
  color: var(--semantic-text);
}
.projects-header,
.project-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
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
  background: var(--semantic-surface);
  border-radius: var(--global-radius-8);
}
.tree-btn {
  background: transparent;
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
}
.row-actions {
  display: inline-flex;
  gap: var(--global-space-8);
}
.conversation-list {
  margin-left: var(--component-tree-indent);
  border-left: 1px solid var(--semantic-divider);
  padding-left: var(--global-space-8);
  display: grid;
  gap: var(--global-space-4);
}
.conversation-item {
  background: var(--semantic-surface);
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  color: var(--semantic-text-muted);
  border-radius: var(--global-radius-8);
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
.bottom {
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface-2);
  padding: var(--global-space-8);
}
.row {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
  color: var(--semantic-text);
  font-size: var(--global-font-size-11);
}
.row a {
  color: inherit;
}
.row.muted {
  color: var(--semantic-text-subtle);
}
.hint {
  margin: 0;
  color: var(--component-toast-info-fg);
  font-size: var(--global-font-size-11);
}
</style>
