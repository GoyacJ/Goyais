import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useRouter } from "vue-router";

import {
  conversationStore,
  ensureConversationRuntime,
  getExecutionStateCounts
} from "@/modules/conversation/store";
import {
  useAutoModelSyncWatcher
} from "@/modules/conversation/views/controllerWatchers";
import { buildExecutionTraceViewModels, type ExecutionTraceViewModel } from "@/modules/conversation/views/processTrace";
import { useExecutionTraceState } from "@/modules/conversation/views/useExecutionTraceState";
import { useRunningActionsView } from "@/modules/conversation/views/useRunningActionsView";
import { useMainScreenActions } from "@/modules/conversation/views/useMainScreenActions";
import { useMainScreenModeling } from "@/modules/conversation/views/useMainScreenModeling";
import { createConversationStreamCoordinator } from "@/modules/conversation/views/streamCoordinator";
import { localizeComposerSuggestionDetails } from "@/modules/conversation/views/composerSuggestionDetails";
import {
  dedupeComposerSuggestions,
  resolveLocalSuggestionLimit,
  resolveSuggestionContext,
  shouldRequestRemoteSuggestions
} from "@/modules/conversation/views/composerSuggestionPolicy";
import { getComposerCatalog, suggestComposerInput } from "@/modules/conversation/services";
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
import { setConversationError } from "@/modules/conversation/store";
import type { ComposerCatalog, ComposerSuggestion, DiffCapability, InspectorTabKey } from "@/shared/types/api";

export function useMainScreenController() {
  const router = useRouter();
  const { t } = useI18n();

  const editingConversationName = ref(false);
  const conversationNameDraft = ref("");
  const inspectorCollapsed = ref(false);
  const projectImportInProgress = ref(false);
  const projectImportFeedback = ref("");
  const projectImportError = ref("");
  const composerCatalog = ref<ComposerCatalog>({
    revision: "",
    commands: [],
    resources: []
  });
  const composerSuggestions = ref<ComposerSuggestion[]>([]);
  const composerSuggesting = ref(false);
  let composerSuggestSequence = 0;

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
        executing: 0
      }
  );
  const queuedCount = computed(() => executionStateCounts.value.queued);
  const pendingCount = computed(() => executionStateCounts.value.pending);
  const executingCount = computed(() => executionStateCounts.value.executing);
  const activeCount = computed(() => pendingCount.value + executingCount.value);
  const baseExecutionTraces = computed<ExecutionTraceViewModel[]>(() => {
    const currentRuntime = runtime.value;
    if (!currentRuntime) {
      return [];
    }
    const tracedExecutions = currentRuntime.executions.filter(
      (execution) => execution.agent_config_snapshot?.show_process_trace ?? true
    );
    return buildExecutionTraceViewModels(currentRuntime.events, tracedExecutions);
  });
  const {
    activeTraceCount,
    executionTraces,
    toggleExecutionTrace
  } = useExecutionTraceState(baseExecutionTraces);
  const { runningActions } = useRunningActionsView(runtime, {
    executionFilter: (execution) => execution.agent_config_snapshot?.show_process_trace ?? true
  });
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

  const composerCatalogRevision = computed(() => composerCatalog.value.revision);

  const actions = useMainScreenActions({
    router,
    activeConversation,
    activeProject,
    runtime,
    modelOptions: modelState.modelOptions,
    composerCatalogRevision,
    inspectorCollapsed,
    editingConversationName,
    conversationNameDraft,
    projectImportInProgress,
    projectImportFeedback,
    projectImportError,
    resolveSemanticModelID: modelState.resolveSemanticModelID
  });

  async function refreshComposerCatalogForActiveConversation(): Promise<void> {
    const conversationId = activeConversation.value?.id?.trim() ?? "";
    if (conversationId === "") {
      composerCatalog.value = {
        revision: "",
        commands: [],
        resources: []
      };
      composerSuggestions.value = [];
      return;
    }
    try {
      composerCatalog.value = await getComposerCatalog(conversationId);
    } catch (error) {
      composerCatalog.value = {
        revision: "",
        commands: [],
        resources: []
      };
      composerSuggestions.value = [];
      setConversationError(String((error as Error)?.message ?? "加载输入目录失败"));
    }
  }

  async function requestComposerSuggestions(input: { draft: string; cursor: number }): Promise<void> {
    const conversationId = activeConversation.value?.id?.trim() ?? "";
    if (conversationId === "") {
      composerSuggestSequence += 1;
      composerSuggesting.value = false;
      composerSuggestions.value = [];
      return;
    }
    const sequence = ++composerSuggestSequence;
    composerSuggesting.value = false;
    const currentDraft = input.draft;
    const cursor = Math.max(0, Math.min(input.cursor, currentDraft.length));
    const context = resolveSuggestionContext(currentDraft, cursor);
    if (context.kind === "none") {
      composerSuggestions.value = [];
      return;
    }
    const localLimit = resolveLocalSuggestionLimit(context, composerCatalog.value, 12);
    const localSuggestions = dedupeComposerSuggestions(
      localizeComposerSuggestionDetails(
        buildLocalComposerSuggestions(currentDraft, cursor, composerCatalog.value, localLimit),
        t
      )
    );
    composerSuggestions.value = localSuggestions;
    if (!shouldRequestRemoteSuggestions(context, localSuggestions.length)) {
      return;
    }
    composerSuggesting.value = true;
    try {
      const response = await suggestComposerInput(conversationId, {
        draft: currentDraft,
        cursor,
        catalog_revision: composerCatalog.value.revision
      });
      if (sequence !== composerSuggestSequence) {
        return;
      }
      const localizedRemoteSuggestions = dedupeComposerSuggestions(
        localizeComposerSuggestionDetails(response.suggestions, t)
      );
      composerSuggestions.value = localizedRemoteSuggestions.length > 0
        ? localizedRemoteSuggestions
        : localSuggestions;
    } catch {
      if (sequence !== composerSuggestSequence) {
        return;
      }
      composerSuggestions.value = localSuggestions;
    } finally {
      if (sequence === composerSuggestSequence) {
        composerSuggesting.value = false;
      }
    }
  }

  function clearComposerSuggestions(): void {
    composerSuggestions.value = [];
  }

  function buildLocalComposerSuggestions(
    draft: string,
    cursor: number,
    catalog: ComposerCatalog,
    limit: number
  ): ComposerSuggestion[] {
    const safeCursor = Math.max(0, Math.min(cursor, draft.length));
    const { tokenStart, tokenEnd, token } = resolveActiveToken(draft, safeCursor);
    const trimmedToken = token.trim();
    if (!trimmedToken.startsWith("@") && !trimmedToken.startsWith("/")) {
      return [];
    }

    if (trimmedToken.startsWith("/")) {
      const commandQuery = trimmedToken.slice(1).toLowerCase();
      return catalog.commands
        .filter((item) => item.name.toLowerCase().includes(commandQuery))
        .slice(0, limit)
        .map((item) => ({
          kind: "command",
          label: `/${item.name}`,
          detail: stringsOrEmpty(item.description),
          insert_text: `/${item.name}`,
          replace_start: tokenStart,
          replace_end: tokenEnd
        }));
    }

    const resourceToken = trimmedToken.slice(1);
    const [resourceTypeRaw, resourceQueryRaw] = resourceToken.split(":", 2);
    const resourceType = resourceTypeRaw?.trim().toLowerCase() ?? "";
    if (!resourceToken.includes(":")) {
      return ["model", "rule", "skill", "mcp", "file"]
        .filter((type) => type.includes(resourceType))
        .slice(0, limit)
        .map((type) => ({
          kind: "resource_type",
          label: `@${type}:`,
          detail: resolveResourceTypeSuggestionDetail(type),
          insert_text: `@${type}:`,
          replace_start: tokenStart,
          replace_end: tokenEnd
        }));
    }

    if (!["model", "rule", "skill", "mcp", "file"].includes(resourceType)) {
      return [];
    }

    const resourceQuery = (resourceQueryRaw ?? "").trim().toLowerCase();
    return catalog.resources
      .filter((resource) => resource.type === resourceType)
      .filter((resource) => {
        if (resourceQuery === "") {
          return true;
        }
        return resource.id.toLowerCase().includes(resourceQuery) || resource.name.toLowerCase().includes(resourceQuery);
      })
      .slice(0, limit)
      .map((resource) => ({
        kind: "resource",
        label: `@${resource.type}:${resource.id}`,
        detail: resolveResourceSuggestionDetail(resource),
        insert_text: `@${resource.type}:${resource.id}`,
        replace_start: tokenStart,
        replace_end: tokenEnd
      }));
  }

  function resolveResourceTypeSuggestionDetail(type: string): string {
    switch (type) {
      case "model":
        return t("conversation.composer.suggestion.type.model");
      case "rule":
        return t("conversation.composer.suggestion.type.rule");
      case "skill":
        return t("conversation.composer.suggestion.type.skill");
      case "mcp":
        return t("conversation.composer.suggestion.type.mcp");
      default:
        return "";
    }
  }

  function resolveResourceSuggestionDetail(resource: { type: string; id: string; name: string }): string {
    if (resource.type === "file") {
      return "";
    }
    const normalizedName = stringsOrEmpty(resource.name);
    const normalizedID = stringsOrEmpty(resource.id);
    if (normalizedName === "" || normalizedName.toLowerCase() === normalizedID.toLowerCase()) {
      return "";
    }
    return normalizedName;
  }

  function stringsOrEmpty(value: string | undefined): string {
    return (value ?? "").trim();
  }

  function resolveActiveToken(draft: string, cursor: number): { tokenStart: number; tokenEnd: number; token: string } {
    let tokenStart = cursor;
    while (tokenStart > 0 && !/\s/.test(draft[tokenStart - 1] ?? "")) {
      tokenStart -= 1;
    }

    let tokenEnd = cursor;
    while (tokenEnd < draft.length && !/\s/.test(draft[tokenEnd] ?? "")) {
      tokenEnd += 1;
    }

    return {
      tokenStart,
      tokenEnd,
      token: draft.slice(tokenStart, cursor)
    };
  }

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
    await refreshComposerCatalogForActiveConversation();
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
        void refreshComposerCatalogForActiveConversation();
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
      void refreshComposerCatalogForActiveConversation();
      streamCoordinator.syncConversationStreams();
    }
  );

  watch(
    () => runtime.value?.draft ?? "",
    (draftValue) => {
      if (draftValue.trim() === "") {
        clearComposerSuggestions();
      }
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
      void refreshComposerCatalogForActiveConversation();
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

  return {
    ...actions,
    activeConversation,
    activeCount,
    activeModelId: modelState.activeModelId,
    activeModelLabel: modelState.activeModelLabel,
    activeProject,
    authStore,
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
    activeTraceCount,
    composerCatalog,
    composerSuggesting,
    composerSuggestions,
    requestComposerSuggestions,
    clearComposerSuggestions,
    executionTraces,
    projectImportError,
    projectImportFeedback,
    projectImportInProgress,
    projectStore,
    projectsPage,
    queuedCount,
    runningState,
    runningStateClass,
    runtime,
    runtimeConnectionStatus: workspaceStatus.connectionStatus,
    runtimeHubLabel: workspaceStatus.hubURL,
    runtimeUserDisplayName: workspaceStatus.userDisplayName,
    runningActions,
    toggleExecutionTrace,
    workspaceLabel,
    workspaceStore
  };
}
