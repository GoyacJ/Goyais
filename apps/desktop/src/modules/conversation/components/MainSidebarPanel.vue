<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="top">
      <WorkspaceSwitcherCard
        :workspaces="workspaces"
        :current-workspace-id="currentWorkspaceId"
        :collapsed="collapsed"
        :show-collapse-toggle="true"
        :can-create-workspace="true"
        fallback-label="工作区"
        @switch-workspace="onSwitchWorkspace"
        @create-workspace="openCreateWorkspace"
        @toggle-collapse="collapsed = !collapsed"
      />

      <div class="projects-header">
        <span class="title">
          <AppIcon name="folder" :size="12" />
          <template v-if="!collapsed">项目</template>
        </span>
        <button class="icon-btn tiny" type="button" :disabled="projectImportInProgress" @click="pickDirectory">
          <AppIcon name="plus" :size="12" />
        </button>
      </div>
      <div v-if="!collapsed" class="import-feedback">
        <ToastAlert v-if="projectImportError !== ''" tone="error" :message="projectImportError" />
        <ToastAlert v-else-if="projectImportInProgress" tone="retrying" message="正在导入项目..." />
        <ToastAlert v-else-if="projectImportFeedback !== ''" tone="info" :message="projectImportFeedback" />
      </div>

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

    <WorkspaceCreateModal :open="createWorkspaceOpen" @close="createWorkspaceOpen = false" @submit="submitWorkspaceCreate" />
  </aside>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";

import { pickDirectoryPath } from "@/shared/services/directoryPicker";
import AppIcon from "@/shared/ui/AppIcon.vue";
import ToastAlert from "@/shared/ui/ToastAlert.vue";
import type { Conversation, Project, Workspace, WorkspaceMode } from "@/shared/types/api";
import WorkspaceCreateModal from "@/shared/ui/sidebar/WorkspaceCreateModal.vue";
import UserProfileMenuCard from "@/shared/ui/sidebar/UserProfileMenuCard.vue";
import WorkspaceSwitcherCard from "@/shared/ui/sidebar/WorkspaceSwitcherCard.vue";

const props = defineProps<{
  workspaces: Workspace[];
  currentWorkspaceId: string;
  workspaceMode: WorkspaceMode;
  workspaceName: string;
  userName: string;
  projects: Project[];
  projectsPage: {
    canPrev: boolean;
    canNext: boolean;
    loading: boolean;
  };
  conversationsByProjectId: Record<string, Conversation[]>;
  conversationPageByProjectId: Record<
    string,
    {
      canPrev: boolean;
      canNext: boolean;
      loading: boolean;
    }
  >;
  activeConversationId: string;
  projectImportInProgress: boolean;
  projectImportFeedback: string;
  projectImportError: string;
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
  (event: "paginateProjects", direction: "prev" | "next"): void;
  (event: "paginateConversations", projectId: string, direction: "prev" | "next"): void;
}>();

const collapsed = ref(false);
const createWorkspaceOpen = ref(false);

const projectOpen = reactive<Record<string, boolean>>({});

const currentWorkspaceMode = computed(() => props.workspaceMode);
const userInitial = computed(() => (props.userName || "L").slice(0, 1).toUpperCase());
const userMenuItems = computed(() => {
  const items = [
    {
      key: "settings",
      label: "设置",
      icon: "settings"
    }
  ];
  if (currentWorkspaceMode.value !== "local") {
    items.unshift({ key: "account", label: "账号信息", icon: "circle-user-round" });
  }
  return items;
});
const projectImportInProgress = computed(() => props.projectImportInProgress);
const projectImportFeedback = computed(() => props.projectImportFeedback);
const projectImportError = computed(() => props.projectImportError);

function toggleProject(projectId: string): void {
  projectOpen[projectId] = !isProjectOpen(projectId);
}

function isProjectOpen(projectId: string): boolean {
  return projectOpen[projectId] ?? true;
}

async function pickDirectory(): Promise<void> {
  const directoryPath = await pickDirectoryPath();
  if (!directoryPath) {
    return;
  }
  const normalizedPath = directoryPath.trim();
  if (normalizedPath === "") {
    return;
  }
  emit("importProject", normalizedPath);
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

function submitWorkspaceCreate(payload: { hub_url: string; username: string; password: string }): void {
  emit("createWorkspace", payload);
  createWorkspaceOpen.value = false;
}
</script>

<style scoped src="./MainSidebarPanel.css"></style>
