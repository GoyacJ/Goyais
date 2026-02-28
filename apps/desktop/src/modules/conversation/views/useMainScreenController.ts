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
import { useQueueMessagesView } from "@/modules/conversation/views/useQueueMessagesView";
import { useMainScreenActions } from "@/modules/conversation/views/useMainScreenActions";
import { useMainScreenModeling } from "@/modules/conversation/views/useMainScreenModeling";
import { createConversationStreamCoordinator } from "@/modules/conversation/views/streamCoordinator";
import { resolveConversationUsage } from "@/modules/conversation/views/conversationTokenUsage";
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
import { refreshResourceConfigsByType } from "@/modules/resource/store";
import {
  initializeWorkspaceContext,
  workspaceStore
} from "@/modules/workspace/store";
import { useI18n } from "@/shared/i18n";
import { authStore } from "@/shared/stores/authStore";
import { useWorkspaceStatusSync } from "@/shared/stores/workspaceStatusStore";
import { setConversationError } from "@/modules/conversation/store";
import type { ComposerCatalog, ComposerSuggestion, DiffCapability, ExecutionEvent, InspectorTabKey } from "@/shared/types/api";

type PendingQuestionOptionViewModel = {
  id: string;
  label: string;
  description: string;
};

type PendingExecutionQuestionViewModel = {
  executionId: string;
  messageId: string;
  queueIndex: number;
  questionId: string;
  question: string;
  options: PendingQuestionOptionViewModel[];
  recommendedOptionId: string;
  allowText: boolean;
  required: boolean;
};

export function useMainScreenController() {
  const router = useRouter();
  const { t, locale } = useI18n();

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
    { key: "trace", label: "T" },
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
  const activeConversationTokenUsage = computed(() =>
    resolveConversationUsage(activeConversation.value, runtime.value)
  );
  const conversationTokenUsageById = computed(() => {
    const result: Record<string, { input: number; output: number; total: number }> = {};
    for (const list of Object.values(projectStore.conversationsByProjectId)) {
      for (const conversation of list) {
        result[conversation.id] = resolveConversationUsage(conversation, conversationStore.byConversationId[conversation.id]);
      }
    }
    return result;
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
  const hasConfirmingExecution = computed(() =>
    (runtime.value?.executions ?? []).some((execution) => execution.state === "confirming")
  );
  const pendingQuestions = computed<PendingExecutionQuestionViewModel[]>(() => {
    const currentRuntime = runtime.value;
    if (!currentRuntime) {
      return [];
    }
    const executionByID = new Map(currentRuntime.executions.map((execution) => [execution.id, execution]));
    const pendingByExecution = new Map<string, PendingExecutionQuestionViewModel>();

    for (const event of currentRuntime.events) {
      const pending = toPendingQuestionFromEvent(event, executionByID);
      if (pending) {
        pendingByExecution.set(pending.executionId, pending);
        continue;
      }
      if (isPendingQuestionResolvedEvent(event)) {
        const executionID = event.execution_id.trim();
        if (executionID === "") {
          continue;
        }
        const questionID = stringOrEmpty(event.payload.question_id);
        if (questionID === "") {
          pendingByExecution.delete(executionID);
          continue;
        }
        const current = pendingByExecution.get(executionID);
        if (current && current.questionId === questionID) {
          pendingByExecution.delete(executionID);
        }
      }
    }

    return [...pendingByExecution.values()]
      .filter((item) => executionByID.get(item.executionId)?.state === "awaiting_input")
      .sort((left, right) => left.queueIndex - right.queueIndex);
  });
  const {
    queuedMessages,
    visibleMessages,
    visibleTraceExecutionIds
  } = useQueueMessagesView(runtime);
  const baseExecutionTraces = computed<ExecutionTraceViewModel[]>(() => {
    const currentRuntime = runtime.value;
    if (!currentRuntime) {
      return [];
    }
    const tracedExecutions = currentRuntime.executions.filter(
      (execution) => execution.agent_config_snapshot?.show_process_trace ?? true
    );
    return buildExecutionTraceViewModels(currentRuntime.events, tracedExecutions, locale.value);
  });
  const visibleExecutionTraces = computed<ExecutionTraceViewModel[]>(() =>
    baseExecutionTraces.value.filter((trace) => visibleTraceExecutionIds.value.has(trace.executionId))
  );
  const {
    activeTraceCount,
    executionTraces,
    selectedExecutionTrace,
    selectedTraceExecutionId,
    selectExecutionTrace
  } = useExecutionTraceState(visibleExecutionTraces);
  const { runningActions } = useRunningActionsView(runtime, {
    locale,
    executionFilter: (execution) =>
      (execution.agent_config_snapshot?.show_process_trace ?? true) &&
      visibleTraceExecutionIds.value.has(execution.id)
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

  function selectTraceInInspector(executionId: string): void {
    selectExecutionTrace(executionId);
    actions.openInspectorTab("trace");
    inspectorCollapsed.value = false;
  }

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
    activeConversationTokenUsage,
    activeCount,
    hasConfirmingExecution,
    pendingQuestions,
    activeModelId: modelState.activeModelId,
    activeModelLabel: modelState.activeModelLabel,
    activeProject,
    authStore,
    conversationNameDraft,
    conversationTokenUsageById,
    conversationPageByProjectId,
    editingConversationName,
    executingCount,
    visibleMessages,
    queuedMessages,
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
    selectedExecutionTrace,
    selectedTraceExecutionId,
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
    selectTraceInInspector,
    workspaceLabel,
    workspaceStore
  };
}

function toPendingQuestionFromEvent(
  event: ExecutionEvent,
  executionByID: Map<string, { id: string; message_id: string; queue_index: number }>
): PendingExecutionQuestionViewModel | null {
  if (event.type !== "thinking_delta") {
    return null;
  }
  if (stringOrEmpty(event.payload.stage) !== "run_user_question_needed") {
    return null;
  }
  const executionID = event.execution_id.trim();
  if (executionID === "") {
    return null;
  }
  const execution = executionByID.get(executionID);
  if (!execution) {
    return null;
  }
  const questionID = stringOrEmpty(event.payload.question_id);
  const question = stringOrEmpty(event.payload.question);
  if (questionID === "" || question === "") {
    return null;
  }
  const options = normalizePendingQuestionOptions(event.payload.options);
  const allowTextRaw = event.payload.allow_text;
  const requiredRaw = event.payload.required;
  const allowText = typeof allowTextRaw === "boolean" ? allowTextRaw : true;
  const required = typeof requiredRaw === "boolean" ? requiredRaw : true;
  const recommendedOptionId = stringOrEmpty(event.payload.recommended_option_id);

  if (!allowText && options.length === 0) {
    return null;
  }

  return {
    executionId: executionID,
    messageId: execution.message_id,
    queueIndex: execution.queue_index,
    questionId: questionID,
    question,
    options,
    recommendedOptionId,
    allowText,
    required
  };
}

function isPendingQuestionResolvedEvent(event: ExecutionEvent): boolean {
  return event.type === "thinking_delta" && stringOrEmpty(event.payload.stage) === "run_user_question_resolved";
}

function normalizePendingQuestionOptions(raw: unknown): PendingQuestionOptionViewModel[] {
  if (!Array.isArray(raw)) {
    return [];
  }
  const options: PendingQuestionOptionViewModel[] = [];
  for (const item of raw) {
    if (typeof item === "string") {
      const label = item.trim();
      if (label === "") {
        continue;
      }
      options.push({
        id: `option_${options.length + 1}`,
        label,
        description: ""
      });
      continue;
    }
    if (typeof item !== "object" || item === null) {
      continue;
    }
    const entry = item as Record<string, unknown>;
    const label = stringOrEmpty(entry.label);
    if (label === "") {
      continue;
    }
    const id = stringOrEmpty(entry.id) || `option_${options.length + 1}`;
    options.push({
      id,
      label,
      description: stringOrEmpty(entry.description)
    });
  }
  return options;
}

function stringOrEmpty(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}
