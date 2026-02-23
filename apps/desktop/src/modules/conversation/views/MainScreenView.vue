<template>
  <MainShell>
    <template #sidebar>
      <MainSidebarPanel
        :workspaces="workspaceStore.workspaces"
        :current-workspace-id="workspaceStore.currentWorkspaceId"
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

            <button class="icon-btn" type="button" @click="startEditConversationName">
              <AppIcon name="pencil" :size="12" />
            </button>
          </div>
        </template>

        <template #right>
          <div class="right">
            <span class="state">{{ runningState }}</span>
            <span :class="connectionClass">{{ connectionState }}</span>
          </div>
        </template>
      </Topbar>
    </template>

    <template #main>
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

        <div class="inspector-slot" :class="{ collapsed: inspectorCollapsed }">
          <MainInspectorPanel
            v-if="!inspectorCollapsed"
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
      </div>
    </template>

    <template #footer>
      <HubStatusBar />
    </template>
  </MainShell>
</template>

<script setup lang="ts">
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
  authStore,
  changeInspectorTab,
  commitDiff,
  connectionClass,
  connectionState,
  conversationNameDraft,
  conversationPageByProjectId,
  createWorkspace,
  deleteConversationById,
  deleteProjectById,
  discardDiff,
  editingConversationName,
  exportConversation,
  exportPatch,
  importProjectDirectory,
  inspectorCollapsed,
  inspectorTabs,
  nonGitCapability,
  onConversationNameInput,
  openAccount,
  openInspectorTab,
  openSettings,
  paginateConversations,
  paginateProjects,
  placeholder,
  projectStore,
  projectImportError,
  projectImportFeedback,
  projectImportInProgress,
  projectsPage,
  queuedCount,
  rollbackMessage,
  runningState,
  runtime,
  saveConversationName,
  selectConversation,
  sendMessage,
  startEditConversationName,
  stopExecution,
  switchWorkspace,
  updateDraft,
  updateMode,
  updateModel,
  workspaceLabel,
  workspaceStore
} = useMainScreenController();

</script>

<style scoped src="./MainScreenView.css"></style>
