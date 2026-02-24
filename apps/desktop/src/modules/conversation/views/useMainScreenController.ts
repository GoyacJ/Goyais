import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useRouter } from "vue-router";

import { attachConversationStream, detachConversationStream, ensureConversationRuntime } from "@/modules/conversation/store";
import { useMainScreenActions } from "@/modules/conversation/views/useMainScreenActions";
import { useMainScreenModeling } from "@/modules/conversation/views/useMainScreenModeling";
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

  const placeholder = computed(() => t("conversation.placeholderInput"));
  const queuedCount = computed(() => runtime.value?.executions.filter((item) => item.state === "queued").length ?? 0);
  const activeCount = computed(() => runtime.value?.executions.filter((item) => item.state === "executing").length ?? 0);
  const runningState = computed(() => (activeCount.value > 0 ? "running" : "idle"));

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

  onMounted(async () => {
    await initializeWorkspaceContext();
    await Promise.all([refreshProjects(), refreshResourceConfigsByType("model")]);
  });

  onUnmounted(() => {
    const activeId = projectStore.activeConversationId;
    if (activeId) {
      detachConversationStream(activeId);
    }
  });

  watch(
    () => projectStore.activeProjectId,
    async () => {
      await refreshConversationsForActiveProject();
    }
  );

  watch(
    () => projectStore.activeConversationId,
    (nextId, prevId) => {
      if (prevId) {
        detachConversationStream(prevId);
      }
      if (!nextId) {
        return;
      }
      const conversation = (projectStore.conversationsByProjectId[projectStore.activeProjectId] ?? []).find((item) => item.id === nextId);
      if (!conversation) {
        return;
      }
      const workspaceToken = workspaceStore.currentWorkspaceId ? authStore.tokensByWorkspaceId[workspaceStore.currentWorkspaceId] : undefined;
      attachConversationStream(conversation, workspaceToken);
    }
  );

  watch(
    () => workspaceStore.currentWorkspaceId,
    () => {
      projectImportInProgress.value = false;
      projectImportFeedback.value = "";
      projectImportError.value = "";
      void refreshResourceConfigsByType("model");
    }
  );

  const autoModelSyncingConversationID = ref("");
  watch(
    [() => activeConversation.value?.id ?? "", () => activeCount.value, () => modelState.modelOptions.value],
    async ([conversationID, executingCount, options]) => {
      if (conversationID === "" || executingCount > 0 || options.length === 0) {
        return;
      }
      if (autoModelSyncingConversationID.value === conversationID) {
        return;
      }
      const currentModelID = modelState.resolveSemanticModelID(runtime.value?.modelId ?? activeConversation.value?.model_id ?? "");
      if (currentModelID !== "" && options.some((item) => item.value === currentModelID)) {
        return;
      }
      const targetModelID = options[0]?.value ?? "";
      if (targetModelID === "") {
        return;
      }
      autoModelSyncingConversationID.value = conversationID;
      try {
        await actions.updateModel(targetModelID);
      } finally {
        autoModelSyncingConversationID.value = "";
      }
    },
    { deep: true }
  );

  watch(
    () => runtime.value?.events.length ?? 0,
    () => {
      const events = runtime.value?.events ?? [];
      const latest = events[events.length - 1];
      if (!latest) {
        return;
      }

      if (latest.type === "confirmation_required") {
        riskConfirm.value = {
          open: true,
          executionId: latest.execution_id,
          summary: typeof latest.payload.summary === "string" ? latest.payload.summary : "高风险操作需要确认",
          preview: typeof latest.payload.preview === "string" ? latest.payload.preview : ""
        };
        return;
      }

      if (latest.type === "confirmation_resolved" || latest.type === "execution_done" || latest.type === "execution_error" || latest.type === "execution_stopped") {
        if (latest.execution_id === riskConfirm.value.executionId) {
          riskConfirm.value.open = false;
          riskConfirm.value.executionId = "";
        }
      }
    }
  );

  return {
    ...actions,
    activeConversation,
    activeCount,
    activeModelId: modelState.activeModelId,
    activeProject,
    authStore,
    connectionClass,
    connectionState,
    conversationNameDraft,
    conversationPageByProjectId,
    editingConversationName,
    inspectorCollapsed,
    inspectorTabs,
    modelOptions: modelState.modelOptions,
    nonGitCapability,
    placeholder,
    projectImportError,
    projectImportFeedback,
    projectImportInProgress,
    projectStore,
    projectsPage,
    queuedCount,
    riskConfirm,
    runningState,
    runtime,
    workspaceLabel,
    workspaceStore
  };
}
