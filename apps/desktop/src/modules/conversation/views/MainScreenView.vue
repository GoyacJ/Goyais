<template>
  <div class="screen">
    <MainSidebarPanel
      :workspaces="workspaceStore.workspaces"
      :current-workspace-id="workspaceStore.currentWorkspaceId"
      :workspace-name="workspaceLabel"
      :workspace-mode="workspaceStore.mode"
      :user-name="authStore.me?.display_name ?? 'local'"
      :projects="projectStore.projects"
      :conversations-by-project-id="projectStore.conversationsByProjectId"
      :active-conversation-id="projectStore.activeConversationId"
      @switch-workspace="switchWorkspace"
      @create-workspace="createWorkspace"
      @import-project="importProjectDirectory"
      @add-conversation="addConversationByPrompt"
      @delete-project="deleteProjectById"
      @export-conversation="exportConversation"
      @delete-conversation="deleteConversationById"
      @select-conversation="selectConversation"
      @open-account="openAccount"
      @open-settings="openSettings"
    />

    <section class="content">
      <header class="top-header">
        <div class="left">
          <strong>{{ activeProject?.name ?? 'Project' }}</strong>
          <span>/</span>

          <template v-if="editingConversationName">
            <input
              class="title-input"
              :value="conversationNameDraft"
              @input="onConversationNameInput"
              @keydown.enter="saveConversationName"
              @blur="saveConversationName"
            />
          </template>
          <strong v-else>{{ activeConversation?.name ?? 'Conversation' }}</strong>

          <button class="icon-btn" type="button" @click="startEditConversationName">
            <AppIcon name="pencil" :size="12" />
          </button>
        </div>

        <div class="right">
          <span class="state">{{ runningState }}</span>
          <span :class="connectionClass">{{ connectionState }}</span>
        </div>
      </header>

      <div class="main-body">
        <MainConversationPanel
          :messages="runtime?.messages ?? []"
          :queued-count="queuedCount"
          :has-active-execution="activeCount > 0"
          :draft="runtime?.draft ?? ''"
          :mode="runtime?.mode ?? 'agent'"
          :model-id="runtime?.modelId ?? 'gpt-4.1'"
          :placeholder="placeholder"
          @update:draft="updateDraft"
          @update:mode="updateMode"
          @update:model="updateModel"
          @send="sendMessage"
          @stop="stopExecution"
          @rollback="rollbackMessage"
        />

        <MainInspectorPanel
          :diff="runtime?.diff ?? []"
          :capability="runtime?.diffCapability ?? nonGitCapability"
          :queued-count="queuedCount"
          :active-count="activeCount"
          :model-id="runtime?.modelId ?? 'gpt-4.1'"
          :active-tab="runtime?.inspectorTab ?? 'diff'"
          @change-tab="changeInspectorTab"
          @commit="commitDiff"
          @discard="discardDiff"
          @export-patch="exportPatch"
        />
      </div>

      <HubStatusBar />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";

import MainConversationPanel from "@/modules/conversation/components/MainConversationPanel.vue";
import MainInspectorPanel from "@/modules/conversation/components/MainInspectorPanel.vue";
import MainSidebarPanel from "@/modules/conversation/components/MainSidebarPanel.vue";
import {
  commitLatestDiff,
  discardLatestDiff,
  ensureConversationRuntime,
  rollbackConversationToMessage,
  setConversationDraft,
  setConversationInspectorTab,
  setConversationMode,
  setConversationModel,
  stopConversationExecution,
  submitConversationMessage
} from "@/modules/conversation/store";
import {
  addConversation,
  deleteConversation,
  deleteProject,
  exportConversationById,
  importProjectByDirectory,
  projectStore,
  refreshConversationsForActiveProject,
  refreshProjects,
  renameConversationById,
  setActiveConversation,
  setActiveProject
} from "@/modules/project/store";
import { createRemoteConnection } from "@/modules/workspace/services";
import {
  initializeWorkspaceContext,
  setWorkspaceConnection,
  switchWorkspaceContext,
  upsertWorkspace,
  workspaceStore
} from "@/modules/workspace/store";
import { useI18n } from "@/shared/i18n";
import { authStore, setWorkspaceToken } from "@/shared/stores/authStore";
import AppIcon from "@/shared/ui/AppIcon.vue";
import HubStatusBar from "@/shared/ui/HubStatusBar.vue";
import type { DiffCapability, InspectorTabKey } from "@/shared/types/api";

const router = useRouter();
const { t } = useI18n();

const editingConversationName = ref(false);
const conversationNameDraft = ref("");

const nonGitCapability: DiffCapability = {
  can_commit: false,
  can_discard: false,
  can_export_patch: true,
  reason: "Non-Git project: commit and discard are disabled"
};

const activeProject = computed(() =>
  projectStore.projects.find((item) => item.id === projectStore.activeProjectId)
);

const activeConversation = computed(() =>
  (projectStore.conversationsByProjectId[projectStore.activeProjectId] ?? []).find(
    (item) => item.id === projectStore.activeConversationId
  )
);

const runtime = computed(() => {
  const conversation = activeConversation.value;
  const project = activeProject.value;
  if (!conversation || !project) {
    return undefined;
  }
  return ensureConversationRuntime(conversation, project.is_git);
});

const placeholder = computed(() => t("conversation.placeholderInput"));
const queuedCount = computed(() => runtime.value?.executions.filter((item) => item.state === "queued").length ?? 0);
const activeCount = computed(() => runtime.value?.executions.filter((item) => item.state === "executing").length ?? 0);
const runningState = computed(() => (activeCount.value > 0 ? "running" : "idle"));
const workspaceLabel = computed(() => workspaceStore.workspaces.find((item) => item.id === workspaceStore.currentWorkspaceId)?.name ?? "本地工作区");

const connectionState = computed(() => {
  if (workspaceStore.connectionState === "ready") {
    return "connected";
  }
  if (workspaceStore.connectionState === "loading") {
    return "reconnecting";
  }
  return "disconnected";
});

const connectionClass = computed(() => {
  if (connectionState.value === "connected") {
    return "connected";
  }
  if (connectionState.value === "reconnecting") {
    return "reconnecting";
  }
  return "disconnected";
});

onMounted(async () => {
  await initializeWorkspaceContext();
  await refreshProjects();
});

watch(
  () => projectStore.activeProjectId,
  async () => {
    await refreshConversationsForActiveProject();
  }
);

function updateDraft(value: string): void {
  if (!activeConversation.value) {
    return;
  }
  setConversationDraft(activeConversation.value.id, value);
}

function updateMode(value: "agent" | "plan"): void {
  if (!activeConversation.value) {
    return;
  }
  setConversationMode(activeConversation.value.id, value);
}

function updateModel(value: string): void {
  if (!activeConversation.value) {
    return;
  }
  setConversationModel(activeConversation.value.id, value);
}

function changeInspectorTab(tab: InspectorTabKey): void {
  if (!activeConversation.value) {
    return;
  }
  setConversationInspectorTab(activeConversation.value.id, tab);
}

async function sendMessage(): Promise<void> {
  if (!activeConversation.value || !activeProject.value) {
    return;
  }
  await submitConversationMessage(activeConversation.value, activeProject.value.is_git);
}

async function stopExecution(): Promise<void> {
  if (!activeConversation.value) {
    return;
  }
  await stopConversationExecution(activeConversation.value);
}

async function rollbackMessage(messageId: string): Promise<void> {
  if (!activeConversation.value) {
    return;
  }
  await rollbackConversationToMessage(activeConversation.value.id, messageId);
}

async function importProjectDirectory(repoPath: string): Promise<void> {
  await importProjectByDirectory(repoPath);
}

async function deleteProjectById(projectId: string): Promise<void> {
  await deleteProject(projectId);
}

async function addConversationByPrompt(projectId: string): Promise<void> {
  const project = projectStore.projects.find((item) => item.id === projectId);
  if (!project) {
    return;
  }
  const name = window.prompt("新增对话名称", "New Conversation");
  if (!name) {
    return;
  }
  await addConversation(project, name);
}

async function deleteConversationById(projectId: string, conversationId: string): Promise<void> {
  await deleteConversation(projectId, conversationId);
}

function selectConversation(projectId: string, conversationId: string): void {
  setActiveProject(projectId);
  setActiveConversation(conversationId);
}

async function exportConversation(conversationId: string): Promise<void> {
  const markdown = await exportConversationById(conversationId);
  if (!markdown) {
    return;
  }

  const blob = new Blob([markdown], { type: "text/markdown;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `${conversationId}.md`;
  link.click();
  URL.revokeObjectURL(url);
}

async function createWorkspace(payload: { hub_url: string; username: string; password: string }): Promise<void> {
  const result = await createRemoteConnection(payload);
  upsertWorkspace(result.workspace);
  setWorkspaceConnection(result.connection);
  if (result.access_token) {
    setWorkspaceToken(result.workspace.id, result.access_token);
  }
  await switchWorkspaceContext(result.workspace.id);
  await refreshProjects();
}

async function switchWorkspace(workspaceId: string): Promise<void> {
  await switchWorkspaceContext(workspaceId);
  await refreshProjects();
}

function openAccount(): void {
  void router.push("/remote/account");
}

function openSettings(): void {
  void router.push("/settings/theme");
}

function startEditConversationName(): void {
  if (!activeConversation.value) {
    return;
  }
  conversationNameDraft.value = activeConversation.value.name;
  editingConversationName.value = true;
}

function onConversationNameInput(event: Event): void {
  conversationNameDraft.value = (event.target as HTMLInputElement).value;
}

async function saveConversationName(): Promise<void> {
  if (!editingConversationName.value || !activeConversation.value || !activeProject.value) {
    editingConversationName.value = false;
    return;
  }

  const name = conversationNameDraft.value.trim();
  editingConversationName.value = false;
  if (name === "" || name === activeConversation.value.name) {
    return;
  }

  await renameConversationById(activeProject.value.id, activeConversation.value.id, name);
}

async function commitDiff(): Promise<void> {
  if (!activeConversation.value) {
    return;
  }
  await commitLatestDiff(activeConversation.value.id);
}

async function discardDiff(): Promise<void> {
  if (!activeConversation.value) {
    return;
  }
  await discardLatestDiff(activeConversation.value.id);
}

function exportPatch(): void {
  window.alert("Patch exported (design stub).");
}
</script>

<style scoped>
.screen {
  height: 100vh;
  display: grid;
  grid-template-columns: auto 1fr;
  gap: var(--global-space-12);
  padding: var(--global-space-12);
  background: var(--semantic-bg);
}

.content {
  padding: var(--global-space-8) var(--global-space-12) 0;
  display: grid;
  grid-template-rows: 40px 1fr 36px;
  gap: var(--global-space-8);
}

.top-header {
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 var(--global-space-12);
}

.left,
.right {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-8);
}

.left span,
.right {
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-12);
}

.left strong {
  color: var(--semantic-text);
  font-size: var(--global-font-size-14);
}

.state {
  color: var(--semantic-warning);
}

.connected {
  color: var(--semantic-success);
}

.reconnecting {
  color: var(--semantic-warning);
}

.disconnected {
  color: var(--semantic-danger);
}

.main-body {
  display: grid;
  grid-template-columns: 1fr 340px;
  gap: var(--global-space-12);
  min-height: 0;
}

.icon-btn {
  border: 0;
  background: transparent;
  color: var(--semantic-text-subtle);
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.title-input {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  color: var(--semantic-text);
  height: 26px;
  padding: 0 var(--global-space-8);
  min-width: 220px;
}
</style>
