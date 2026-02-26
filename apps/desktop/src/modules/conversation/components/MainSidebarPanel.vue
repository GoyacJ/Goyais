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
        <button class="icon-btn tiny" type="button" :disabled="projectImportInProgress || !supportsDirectoryImport" @click="pickDirectory">
          <AppIcon name="plus" :size="12" />
        </button>
      </div>
      <div v-if="!collapsed && connectionState === 'auth_required'" class="import-feedback">
        <button
          class="auth-login-btn"
          type="button"
          @click="requestWorkspaceLogin"
        >
          重新登录
        </button>
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
import { computed, onBeforeUnmount, reactive, ref, watch } from "vue";

import { isRuntimeCapabilitySupported } from "@/shared/runtime";
import { pickDirectoryPath } from "@/shared/services/directoryPicker";
import { dismissToastByKey, showToast } from "@/shared/stores/toastStore";
import AppIcon from "@/shared/ui/AppIcon.vue";
import type { ConnectionState } from "@/shared/stores/workspaceStore";
import type { Conversation, Project, Workspace, WorkspaceMode } from "@/shared/types/api";
import WorkspaceCreateModal from "@/shared/ui/sidebar/WorkspaceCreateModal.vue";
import UserProfileMenuCard from "@/shared/ui/sidebar/UserProfileMenuCard.vue";
import WorkspaceSwitcherCard from "@/shared/ui/sidebar/WorkspaceSwitcherCard.vue";

const PROJECT_IMPORT_TOAST_KEY = "project-import-status";
const AUTH_REQUIRED_TOAST_KEY = "workspace-auth-required";

const props = defineProps<{
  workspaces: Workspace[];
  currentWorkspaceId: string;
  connectionState?: ConnectionState;
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
  (event: "loginWorkspace", payload: { workspaceId: string; username?: string; password?: string; token?: string }): void;
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
const supportsDirectoryImport = isRuntimeCapabilitySupported("supportsDirectoryImport");

watch(
  () => [projectImportInProgress.value, projectImportFeedback.value, projectImportError.value] as const,
  ([inProgress, feedback, error]) => {
    if (inProgress) {
      showToast({
        key: PROJECT_IMPORT_TOAST_KEY,
        tone: "retrying",
        message: "正在导入项目...",
        persistent: true
      });
      return;
    }
    if (error !== "") {
      showToast({
        key: PROJECT_IMPORT_TOAST_KEY,
        tone: "error",
        message: error
      });
      return;
    }
    if (feedback !== "") {
      showToast({
        key: PROJECT_IMPORT_TOAST_KEY,
        tone: "info",
        message: feedback
      });
      return;
    }
    dismissToastByKey(PROJECT_IMPORT_TOAST_KEY);
  },
  { immediate: true }
);

watch(
  () => props.connectionState,
  (connectionState) => {
    if (connectionState === "auth_required") {
      showToast({
        key: AUTH_REQUIRED_TOAST_KEY,
        tone: "warning",
        message: "远程工作区鉴权已失效，请重新登录。",
        persistent: true
      });
      return;
    }
    dismissToastByKey(AUTH_REQUIRED_TOAST_KEY);
  },
  { immediate: true }
);

function toggleProject(projectId: string): void {
  projectOpen[projectId] = !isProjectOpen(projectId);
}

function isProjectOpen(projectId: string): boolean {
  return projectOpen[projectId] ?? true;
}

async function pickDirectory(): Promise<void> {
  if (!supportsDirectoryImport) {
    showToast({
      tone: "warning",
      message: "当前运行环境不支持目录导入"
    });
    return;
  }

  try {
    const directoryPath = await pickDirectoryPath();
    if (!directoryPath) {
      showToast({
        tone: "warning",
        message: "未选择目录或目录读取失败，请重试"
      });
      return;
    }
    const normalizedPath = directoryPath.trim();
    if (normalizedPath === "") {
      showToast({
        tone: "warning",
        message: "未选择目录或目录读取失败，请重试"
      });
      return;
    }
    emit("importProject", normalizedPath);
  } catch {
    showToast({
      tone: "warning",
      message: "目录选择失败，请重试"
    });
  }
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

function requestWorkspaceLogin(): void {
  const workspaceId = props.currentWorkspaceId.trim();
  if (workspaceId === "") {
    return;
  }

  const username = window.prompt("用户名（可空）", "") ?? "";
  const password = window.prompt("密码（可空）", "") ?? "";
  const token = window.prompt("Token（可空）", "") ?? "";

  emit("loginWorkspace", {
    workspaceId,
    username: username.trim() === "" ? undefined : username.trim(),
    password: password.trim() === "" ? undefined : password,
    token: token.trim() === "" ? undefined : token.trim()
  });
}

onBeforeUnmount(() => {
  dismissToastByKey(PROJECT_IMPORT_TOAST_KEY);
  dismissToastByKey(AUTH_REQUIRED_TOAST_KEY);
});
</script>

<style scoped src="./MainSidebarPanel.css"></style>
