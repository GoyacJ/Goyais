import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useRouter } from "vue-router";

import {
  conversationStore,
  ensureConversationRuntime,
  getExecutionStateCounts
} from "@/modules/conversation/store";
import {
  useAutoModelSyncWatcher,
  useRiskConfirmWatcher
} from "@/modules/conversation/views/controllerWatchers";
import { useMainScreenActions } from "@/modules/conversation/views/useMainScreenActions";
import { useMainScreenModeling } from "@/modules/conversation/views/useMainScreenModeling";
import { createConversationStreamCoordinator } from "@/modules/conversation/views/streamCoordinator";
import {
  projectStore,
  refreshConversationsForActiveProject,
  refreshProjects
} from "@/modules/project/store";
import { refreshResourceConfigsByType, resourceStore } from "@/modules/resource/store";
import {
  initializeWorkspaceContext,
  workspaceStore
} from "@/modules/workspace/store";
import { useI18n } from "@/shared/i18n";
import { authStore } from "@/shared/stores/authStore";
import { useWorkspaceStatusSync } from "@/shared/stores/workspaceStatusStore";
import type { DiffCapability, InspectorTabKey } from "@/shared/types/api";

export function useMainScreenController() {
  const router = useRouter();
  const { t } = useI18n();

  const editingConversationName = ref(false);
  const conversationNameDraft = ref("");
  const inspectorCollapsed = ref(false);
  const riskConfirm = ref({
    open: false,
    executionId: "",
    summary: "",
    preview: ""
  });
  const projectImportInProgress = ref(false);
  const projectImportFeedback = ref("");
  const projectImportError = ref("");

  const inspectorTabs: Array<{ key: InspectorTabKey; label: string }> = [
    { key: "diff", label: "D" },
    { key: "run", label: "R" },
    { key: "files", label: "F" },
    { key: "risk", label: "!" }
  ];

  const nonGitCapability: DiffCapability = {
    can_commit: false,
    can_discard: false,
    can_export_patch: true,
    reason: "Non-Git project: commit and discard are disabled"
  };

  const activeProject = computed(() => projectStore.projects.find((item) => item.id === projectStore.activeProjectId));
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

  const workspaceStatus = useWorkspaceStatusSync({
    conversationId: computed(() => activeConversation.value?.id ?? "")
  });

  const placeholder = computed(() => t("conversation.placeholderInput"));
  const executionStateCounts = computed(() =>
    runtime.value
      ? getExecutionStateCounts(runtime.value)
      : {
        queued: 0,
        pending: 0,
        executing: 0,
        confirming: 0
      }
  );
  const queuedCount = computed(() => executionStateCounts.value.queued);
  const pendingCount = computed(() => executionStateCounts.value.pending);
  const executingCount = computed(() => executionStateCounts.value.executing);
  const confirmingCount = computed(() => executionStateCounts.value.confirming);
  const activeCount = computed(() => pendingCount.value + executingCount.value + confirmingCount.value);
  const runningState = computed(() => workspaceStatus.conversationStatus.value);
  const runningStateClass = computed(() => {
    switch (runningState.value) {
      case "running":
        return "running";
      case "queued":
        return "queued";
      case "done":
        return "done";
      case "error":
        return "error";
      default:
        return "stopped";
    }
  });

  const workspaceLabel = computed(
    () => workspaceStore.workspaces.find((item) => item.id === workspaceStore.currentWorkspaceId)?.name ?? "本地工作区"
  );
  const projectsPage = computed(() => ({
    canPrev: projectStore.projectsPage.backStack.length > 0,
    canNext: projectStore.projectsPage.nextCursor !== null,
    loading: projectStore.projectsPage.loading
  }));
  const conversationPageByProjectId = computed(() => {
    const result: Record<string, { canPrev: boolean; canNext: boolean; loading: boolean }> = {};
    for (const project of projectStore.projects) {
      const page = projectStore.conversationPagesByProjectId[project.id];
      result[project.id] = {
        canPrev: (page?.backStack.length ?? 0) > 0,
        canNext: page?.nextCursor !== null && page?.nextCursor !== undefined,
        loading: page?.loading ?? false
      };
    }
    return result;
  });

  const modelState = useMainScreenModeling({
    activeProject,
    activeConversation,
    runtime
  });

  const actions = useMainScreenActions({
    router,
    activeConversation,
    activeProject,
    runtime,
    modelOptions: modelState.modelOptions,
    inspectorCollapsed,
    riskConfirm,
    editingConversationName,
    conversationNameDraft,
    projectImportInProgress,
    projectImportFeedback,
    projectImportError,
    resolveSemanticModelID: modelState.resolveSemanticModelID
  });

  const streamCoordinator = createConversationStreamCoordinator({
    projects: () => projectStore.projects,
    conversationsByProjectId: () => projectStore.conversationsByProjectId,
    activeConversationId: () => projectStore.activeConversationId,
    resolveToken: () =>
      workspaceStore.currentWorkspaceId ? authStore.tokensByWorkspaceId[workspaceStore.currentWorkspaceId] : undefined
  });

  onMounted(async () => {
    await initializeWorkspaceContext();
    await Promise.all([refreshProjects(), refreshResourceConfigsByType("model")]);
    streamCoordinator.syncConversationStreams();
  });

  onUnmounted(() => {
    streamCoordinator.clearStreams();
  });

  watch(
    () => projectStore.activeProjectId,
    async () => {
      await refreshConversationsForActiveProject();
      streamCoordinator.syncConversationStreams();
    }
  );

  watch(
    () => projectStore.activeConversationId,
    (nextId) => {
      if (!nextId) {
        streamCoordinator.syncConversationStreams();
        return;
      }
      const context = streamCoordinator.findConversationContextById(nextId);
      if (!context) {
        streamCoordinator.syncConversationStreams();
        return;
      }
      ensureConversationRuntime(context.conversation, context.isGitProject);
      void streamCoordinator.hydrateConversationDetail(context, true);
      streamCoordinator.syncConversationStreams();
    }
  );

  watch(
    () => projectStore.conversationsByProjectId,
    () => {
      streamCoordinator.syncConversationStreams();
    },
    { deep: true }
  );

  watch(
    () =>
      Object.entries(conversationStore.byConversationId)
        .map(([conversationId, runtimeValue]) =>
          `${conversationId}:${runtimeValue.executions.map((execution) => execution.state).join(",")}`
        )
        .sort()
        .join("|"),
    () => {
      streamCoordinator.syncConversationStreams();
    }
  );

  watch(
    () => workspaceStore.currentWorkspaceId,
    () => {
      streamCoordinator.clearStreams();
      projectImportInProgress.value = false;
      projectImportFeedback.value = "";
      projectImportError.value = "";
      void refreshResourceConfigsByType("model");
      streamCoordinator.syncConversationStreams();
    }
  );

  useAutoModelSyncWatcher({
    activeConversation,
    activeCount,
    modelOptions: modelState.modelOptions,
    resolveSemanticModelID: modelState.resolveSemanticModelID,
    runtime,
    updateModel: actions.updateModel
  });

  useRiskConfirmWatcher({
    runtime,
    riskConfirm
  });

  return {
    ...actions,
    activeConversation,
    activeCount,
    activeModelId: modelState.activeModelId,
    activeProject,
    authStore,
    confirmingCount,
    conversationNameDraft,
    conversationPageByProjectId,
    editingConversationName,
    executingCount,
    inspectorCollapsed,
    inspectorTabs,
    modelOptions: modelState.modelOptions,
    nonGitCapability,
    pendingCount,
    placeholder,
    projectImportError,
    projectImportFeedback,
    projectImportInProgress,
    projectStore,
    projectsPage,
    queuedCount,
    riskConfirm,
    runningState,
    runningStateClass,
    runtime,
    runtimeConnectionStatus: workspaceStatus.connectionStatus,
    runtimeHubLabel: workspaceStatus.hubURL,
    runtimeUserDisplayName: workspaceStatus.userDisplayName,
    workspaceLabel,
    workspaceStore
  };
}
