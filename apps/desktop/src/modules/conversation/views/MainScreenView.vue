<template>
  <div class="screen">
    <MainSidebarPanel
      :workspace-name="workspaceLabel"
      :workspace-mode="workspaceStore.mode"
      :projects="projectStore.projects"
      :conversations-by-project-id="projectStore.conversationsByProjectId"
      :active-conversation-id="projectStore.activeConversationId"
      @add-project="addProjectByPrompt"
      @delete-project="deleteProjectById"
      @add-conversation="addConversationByPrompt"
      @delete-conversation="deleteConversationById"
      @select-conversation="selectConversation"
    />
    <section class="content">
      <header class="top-header">
        <div class="left">
          <strong>{{ activeProject?.name ?? "Project" }}</strong>
          <span>/</span>
          <strong>{{ activeConversation?.name ?? "Conversation" }}</strong>
          <IconSymbol name="edit" :size="12" />
        </div>
        <div class="right">
          <span class="state">{{ runningState }}</span>
          <span class="connected">{{ runtime?.status ?? "connected" }}</span>
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
          @commit="commitDiff"
          @discard="discardDiff"
          @export-patch="exportPatch"
        />
      </div>
      <footer class="status-bar">
        <span>Hub: {{ workspaceStore.workspaces.find((item) => item.id === workspaceStore.currentWorkspaceId)?.hub_url ?? "https://hub.goyais.local" }}</span>
        <span>{{ authStore.me?.role ?? "Owner" }} · connected</span>
      </footer>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from "vue";

import MainConversationPanel from "@/modules/conversation/components/MainConversationPanel.vue";
import MainInspectorPanel from "@/modules/conversation/components/MainInspectorPanel.vue";
import MainSidebarPanel from "@/modules/conversation/components/MainSidebarPanel.vue";
import {
  commitLatestDiff,
  discardLatestDiff,
  ensureConversationRuntime,
  getConversationRuntime,
  rollbackConversationToMessage,
  setConversationDraft,
  setConversationMode,
  setConversationModel,
  stopConversationExecution,
  submitConversationMessage
} from "@/modules/conversation/store";
import {
  addConversation,
  addProject,
  deleteConversation,
  deleteProject,
  projectStore,
  refreshConversationsForActiveProject,
  refreshProjects,
  setActiveConversation,
  setActiveProject
} from "@/modules/project/store";
import { initializeWorkspaceContext, workspaceStore } from "@/modules/workspace/store";
import { useI18n } from "@/shared/i18n";
import { authStore } from "@/shared/stores/authStore";
import IconSymbol from "@/shared/ui/IconSymbol.vue";
import type { DiffCapability } from "@/shared/types/api";

const { t } = useI18n();

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

function rollbackMessage(messageId: string): void {
  if (!activeConversation.value) {
    return;
  }
  rollbackConversationToMessage(activeConversation.value.id, messageId);
}

async function addProjectByPrompt(): Promise<void> {
  const name = window.prompt("新增项目名称", "New Project");
  if (!name) {
    return;
  }
  await addProject({ name, repo_path: "/workspace/new-project", is_git: true });
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
  grid-template-columns: 320px 1fr;
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
.main-body {
  display: grid;
  grid-template-columns: 1fr 340px;
  gap: var(--global-space-12);
  min-height: 0;
}
.status-bar {
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  color: var(--semantic-text-muted);
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 var(--global-space-12);
  font-size: var(--global-font-size-11);
}
</style>
