<template>
  <MainShell>
    <template #sidebar>
      <MainSidebarPanel
        :workspaces="workspaceStore.workspaces"
        :current-workspace-id="workspaceStore.currentWorkspaceId"
        :connection-state="workspaceStore.connectionState"
        :workspace-name="workspaceLabel"
        :workspace-mode="workspaceStore.mode"
        :user-name="authStore.me?.display_name ?? 'local'"
        :projects="projectStore.projects"
        :projects-page="projectsPage"
        :conversations-by-project-id="projectStore.conversationsByProjectId"
        :conversation-token-usage-by-id="conversationTokenUsageById"
        :conversation-page-by-project-id="conversationPageByProjectId"
        :active-conversation-id="projectStore.activeConversationId"
        :project-import-in-progress="projectImportInProgress"
        :project-import-feedback="projectImportFeedback"
        :project-import-error="projectImportError"
        @switch-workspace="switchWorkspace"
        @create-workspace="createWorkspace"
        @import-project="importProjectDirectory"
        @add-conversation="addConversationByPrompt"
        @delete-project="deleteProjectById"
        @export-conversation="exportConversation"
        @delete-conversation="deleteConversationById"
        @select-conversation="selectConversation"
        @rename-conversation="renameConversation"
        @open-account="openAccount"
        @open-settings="openSettings"
        @paginate-projects="paginateProjects"
        @paginate-conversations="paginateConversations"
        @login-workspace="loginWorkspace"
      />
    </template>

    <template #header>
      <Topbar>
        <template #left>
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
            <span v-if="activeConversation" class="token-usage">
              Token in {{ activeConversationTokenUsage.input }} / out {{ activeConversationTokenUsage.output }} / total {{ activeConversationTokenUsage.total }}
            </span>

            <button v-if="activeConversation" class="icon-btn" type="button" @click="startEditConversationName">
              <AppIcon name="pencil" :size="12" />
            </button>
          </div>
        </template>

        <template #right>
          <div class="right">
            <button class="icon-btn mobile-only" type="button" @click="openInspectorMobile">
              <AppIcon name="panel-right-open" :size="12" />
            </button>
            <span class="state" :class="runningStateClass">{{ runningState }}</span>
          </div>
        </template>
      </Topbar>
    </template>

    <template #main>
      <div class="main-body" :class="{ empty: !activeConversation }">
        <template v-if="activeConversation">
          <MainConversationPanel
            :messages="visibleMessages"
            :queued-messages="queuedMessages"
            :queued-count="queuedCount"
            :pending-count="pendingCount"
            :executing-count="executingCount"
            :has-active-execution="activeCount > 0"
            :has-confirming-execution="hasConfirmingExecution"
            :active-trace-count="activeTraceCount"
            :execution-traces="executionTraces"
            :running-actions="runningActions"
            :pending-questions="pendingQuestions"
            :draft="runtime?.draft ?? ''"
            :mode="runtime?.mode ?? 'default'"
            :model-id="activeModelId"
            :is-model-switching="isSwitchingModel"
            :model-options="modelOptions"
            :placeholder="placeholder"
            :composer-suggestions="composerSuggestions"
            :composer-suggesting="composerSuggesting"
            @update:draft="updateDraft"
            @update:mode="updateMode"
            @update:model="updateModel"
            @request-suggestions="requestComposerSuggestions"
            @clear-suggestions="clearComposerSuggestions"
            @send="sendMessage"
            @stop="stopExecution"
            @remove-queued="removeQueuedMessage"
            @approve="approveExecution"
            @deny="denyExecution"
            @answer-question="answerExecutionQuestion"
            @rollback="rollbackMessage"
            @select-trace="selectTraceInInspector"
          />

          <div class="inspector-slot" :class="{ collapsed: inspectorCollapsed, 'mobile-open': !inspectorCollapsed }">
            <MainInspectorPanel
              v-if="!inspectorCollapsed"
              :change-set="runtime?.changeSet ?? null"
              :capability="runtime?.changeSet?.capability ?? runtime?.diffCapability ?? nonGitCapability"
              :queued-count="queuedCount"
              :pending-count="pendingCount"
              :executing-count="executingCount"
              :run-task-graph="runTaskGraph"
              :run-task-graph-loading="runTaskGraphLoading"
              :run-task-items="runTaskListItems"
              :run-task-list-loading="runTaskListLoading"
              :run-task-list-next-cursor="runTaskListNextCursor"
              :run-task-state-filter="runTaskStateFilter"
              :selected-run-task="selectedRunTask"
              :run-task-detail-loading="runTaskDetailLoading"
              :model-label="activeModelLabel"
              :messages="visibleMessages"
              :executions="runtime?.executions ?? []"
              :events="runtime?.events ?? []"
              :execution-traces="executionTraces"
              :selected-trace-message-id="selectedTraceMessageId"
              :selected-trace-execution-id="selectedTraceExecutionId"
              :active-tab="runtime?.inspectorTab ?? 'diff'"
              @change-tab="changeInspectorTab"
              @select-trace-message="selectTraceMessage"
              @select-trace-execution="selectTraceExecution"
              @commit="openCommitDialog"
              @discard="discardDiff"
              @export-patch="exportPatch"
              @refresh-run-tasks="refreshRunTaskGraph"
              @change-run-task-state-filter="changeRunTaskStateFilter"
              @select-run-task="selectRunTask"
              @load-more-run-tasks="loadMoreRunTasks"
              @control-run-task="controlRunTask"
              @toggle-collapse="inspectorCollapsed = true"
            />

            <aside v-else class="inspector-rail">
              <button class="rail-btn rail-expand" type="button" title="展开 Inspector" @click="inspectorCollapsed = false">
                <AppIcon name="panel-right-open" :size="12" />
              </button>
              <button
                v-for="item in inspectorTabs"
                :key="item.key"
                class="rail-btn"
                :class="{ active: item.key === (runtime?.inspectorTab ?? 'diff') }"
                type="button"
                @click="openInspectorTab(item.key)"
              >
                {{ item.label }}
              </button>
            </aside>
          </div>
          <button
            v-if="!inspectorCollapsed"
            class="inspector-backdrop mobile-only"
            type="button"
            aria-label="关闭 Inspector"
            @click="inspectorCollapsed = true"
          ></button>
        </template>

        <div v-else class="main-empty">
          <p class="main-empty-title">未选择会话</p>
          <p class="main-empty-description">请在左侧会话列表中选择一个会话后开始。</p>
        </div>
      </div>
    </template>

    <template #footer>
      <HubStatusBar
        runtime-mode
        :hub-label="runtimeHubLabel"
        :user-label="runtimeUserDisplayName"
        :connection-status="runtimeConnectionStatus"
      />
    </template>

    <CommitDialog
      :visible="commitDialogVisible"
      :default-message="runtime?.changeSet?.suggested_message.message ?? ''"
      :pending="commitDialogPending"
      @close="closeCommitDialog"
      @confirm="confirmCommitDialog"
    />
  </MainShell>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";

import CommitDialog from "@/modules/session/components/CommitDialog.vue";
import MainConversationPanel from "@/modules/session/components/MainConversationPanel.vue";
import MainInspectorPanel from "@/modules/session/components/MainInspectorPanel.vue";
import MainSidebarPanel from "@/modules/session/components/MainSidebarPanel.vue";
import MainShell from "@/shared/shells/MainShell.vue";
import AppIcon from "@/shared/ui/AppIcon.vue";
import HubStatusBar from "@/shared/ui/HubStatusBar.vue";
import Topbar from "@/shared/ui/Topbar.vue";
import { useMainScreenController } from "@/modules/session/views/useMainScreenController";

const {
  activeConversation,
  activeConversationTokenUsage,
  activeCount,
  activeProject,
  addConversationByPrompt,
  approveExecution,
  authStore,
  changeInspectorTab,
  clearComposerSuggestions,
  commitDiff,
  composerSuggestions,
  composerSuggesting,
  changeRunTaskStateFilter,
  controlRunTask,
  conversationNameDraft,
  conversationTokenUsageById,
  conversationPageByProjectId,
  createWorkspace,
  deleteConversationById,
  deleteProjectById,
  denyExecution,
  answerExecutionQuestion,
  discardDiff,
  editingConversationName,
  executingCount,
  executionTraces,
  exportConversation,
  exportPatch,
  importProjectDirectory,
  isSwitchingModel,
  inspectorCollapsed,
  inspectorTabs,
  loginWorkspace,
  nonGitCapability,
  onConversationNameInput,
  openAccount,
  openInspectorTab,
  openSettings,
  paginateConversations,
  paginateProjects,
  placeholder,
  pendingCount,
  pendingQuestions,
  hasConfirmingExecution,
  projectStore,
  activeTraceCount,
  projectImportError,
  projectImportFeedback,
  projectImportInProgress,
  projectsPage,
  queuedCount,
  queuedMessages,
  rollbackMessage,
  renameConversation,
  removeQueuedMessage,
  runTaskDetailLoading,
  runTaskListItems,
  runTaskListLoading,
  runTaskListNextCursor,
  runTaskStateFilter,
  runningState,
  runningStateClass,
  runningActions,
  runtimeConnectionStatus,
  runtimeHubLabel,
  runtimeUserDisplayName,
  requestComposerSuggestions,
  refreshRunTaskGraph,
  selectTraceInInspector,
  selectTraceMessage,
  selectTraceExecution,
  selectedTraceMessageId,
  selectedTraceExecutionId,
  selectedRunTask,
  runTaskGraph,
  runTaskGraphLoading,
  runtime,
  loadMoreRunTasks,
  selectRunTask,
  visibleMessages,
  saveConversationName,
  selectConversation,
  sendMessage,
  startEditConversationName,
  stopExecution,
  switchWorkspace,
  updateDraft,
  activeModelLabel,
  activeModelId,
  modelOptions,
  updateMode,
  updateModel,
  workspaceLabel,
  workspaceStore
} = useMainScreenController();

onMounted(() => {
  if (typeof window === "undefined") {
    return;
  }
  if (window.matchMedia("(max-width: 768px)").matches) {
    inspectorCollapsed.value = true;
  }
});

const commitDialogVisible = ref(false);
const commitDialogPending = ref(false);

function openInspectorMobile(): void {
  inspectorCollapsed.value = !inspectorCollapsed.value;
}

function openCommitDialog(): void {
  const canCommit = runtime.value?.changeSet?.capability.can_commit ?? false;
  if (!canCommit) {
    return;
  }
  commitDialogVisible.value = true;
}

function closeCommitDialog(): void {
  if (commitDialogPending.value) {
    return;
  }
  commitDialogVisible.value = false;
}

async function confirmCommitDialog(message: string): Promise<void> {
  commitDialogPending.value = true;
  try {
    await commitDiff(message);
    commitDialogVisible.value = false;
  } finally {
    commitDialogPending.value = false;
  }
}

</script>

<style scoped src="./MainScreenView.css"></style>
