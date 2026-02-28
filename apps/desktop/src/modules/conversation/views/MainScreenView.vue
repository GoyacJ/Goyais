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
            :messages="runtime?.messages ?? []"
            :queued-count="queuedCount"
            :pending-count="pendingCount"
            :executing-count="executingCount"
            :has-active-execution="activeCount > 0"
            :has-confirming-execution="hasConfirmingExecution"
            :active-trace-count="activeTraceCount"
            :execution-traces="executionTraces"
            :running-actions="runningActions"
            :draft="runtime?.draft ?? ''"
            :mode="runtime?.mode ?? 'agent'"
            :model-id="activeModelId"
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
            @approve="approveExecution"
            @deny="denyExecution"
            @rollback="rollbackMessage"
            @toggle-trace="toggleExecutionTrace"
          />

          <div class="inspector-slot" :class="{ collapsed: inspectorCollapsed, 'mobile-open': !inspectorCollapsed }">
            <MainInspectorPanel
              v-if="!inspectorCollapsed"
              :diff="runtime?.diff ?? []"
              :capability="runtime?.diffCapability ?? nonGitCapability"
              :queued-count="queuedCount"
              :pending-count="pendingCount"
              :executing-count="executingCount"
              :model-label="activeModelLabel"
              :executions="runtime?.executions ?? []"
              :events="runtime?.events ?? []"
              :active-tab="runtime?.inspectorTab ?? 'diff'"
              @change-tab="changeInspectorTab"
              @commit="commitDiff"
              @discard="discardDiff"
              @export-patch="exportPatch"
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
          <p class="main-empty-title">未选择对话</p>
          <p class="main-empty-description">请在左侧会话列表中选择一个对话后开始。</p>
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

  </MainShell>
</template>

<script setup lang="ts">
import { onMounted } from "vue";

import MainConversationPanel from "@/modules/conversation/components/MainConversationPanel.vue";
import MainInspectorPanel from "@/modules/conversation/components/MainInspectorPanel.vue";
import MainSidebarPanel from "@/modules/conversation/components/MainSidebarPanel.vue";
import MainShell from "@/shared/shells/MainShell.vue";
import AppIcon from "@/shared/ui/AppIcon.vue";
import HubStatusBar from "@/shared/ui/HubStatusBar.vue";
import Topbar from "@/shared/ui/Topbar.vue";
import { useMainScreenController } from "@/modules/conversation/views/useMainScreenController";

const {
  activeConversation,
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
  conversationNameDraft,
  conversationPageByProjectId,
  createWorkspace,
  deleteConversationById,
  deleteProjectById,
  denyExecution,
  discardDiff,
  editingConversationName,
  executingCount,
  executionTraces,
  exportConversation,
  exportPatch,
  importProjectDirectory,
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
  hasConfirmingExecution,
  projectStore,
  activeTraceCount,
  projectImportError,
  projectImportFeedback,
  projectImportInProgress,
  projectsPage,
  queuedCount,
  rollbackMessage,
  runningState,
  runningStateClass,
  runningActions,
  runtimeConnectionStatus,
  runtimeHubLabel,
  runtimeUserDisplayName,
  requestComposerSuggestions,
  toggleExecutionTrace,
  runtime,
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

function openInspectorMobile(): void {
  inspectorCollapsed.value = !inspectorCollapsed.value;
}

</script>

<style scoped src="./MainScreenView.css"></style>
