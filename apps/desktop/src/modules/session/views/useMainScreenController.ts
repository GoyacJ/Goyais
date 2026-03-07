import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useRouter } from "vue-router";

import {
  controlConversationRunTask,
  conversationStore,
  ensureConversationRuntime,
  getRunStateCounts,
  loadConversationRunTaskById,
  loadConversationRunTaskGraph,
  loadConversationRunTasks,
  refreshConversationChangeSet
} from "@/modules/session/store";
import {
  useAutoModelSyncWatcher
} from "@/modules/session/views/controllerWatchers";
import { buildRunTraceViewModels, type RunTraceViewModel } from "@/modules/session/views/processTrace";
import { useRunTraceState } from "@/modules/session/views/useRunTraceState";
import { useRunningActionsView } from "@/modules/session/views/useRunningActionsView";
import { useQueueMessagesView } from "@/modules/session/views/useQueueMessagesView";
import { useMainScreenActions } from "@/modules/session/views/useMainScreenActions";
import { useMainScreenModeling } from "@/modules/session/views/useMainScreenModeling";
import { createConversationStreamCoordinator } from "@/modules/session/views/streamCoordinator";
import { resolveConversationUsage } from "@/modules/session/views/conversationTokenUsage";
import { localizeComposerSuggestionDetails } from "@/modules/session/views/composerSuggestionDetails";
import {
  dedupeComposerSuggestions,
  resolveLocalSuggestionLimit,
  resolveSuggestionContext,
  shouldRequestRemoteSuggestions
} from "@/modules/session/views/composerSuggestionPolicy";
import { getComposerCatalog, suggestComposerInput } from "@/modules/session/services";
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
import { showToast } from "@/shared/stores/toastStore";
import { useWorkspaceStatusSync } from "@/shared/stores/workspaceStatusStore";
import { setConversationError, setConversationModel } from "@/modules/session/store";
import type {
  ChangeSetCapability,
  ComposerCatalog,
  ComposerSuggestion,
  InspectorTabKey,
  RunLifecycleEvent,
  OpenAPIContractComponents
} from "@/shared/types/api";

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

type RunTaskNode = OpenAPIContractComponents["schemas"]["TaskNode"];
type RunTaskState = OpenAPIContractComponents["schemas"]["TaskState"];

const INSPECTOR_ERROR_TOAST_KEY = "session-inspector-error";

export function isSameConversationResponseTarget(expectedConversationId: string, activeConversationId: string): boolean {
  return expectedConversationId.trim() !== "" && expectedConversationId.trim() === activeConversationId.trim();
}

export function shouldApplyRunTaskDetailResponse(
  expectedConversationId: string,
  activeConversationId: string,
  expectedTaskId: string,
  selectedTaskId: string
): boolean {
  return isSameConversationResponseTarget(expectedConversationId, activeConversationId) &&
    expectedTaskId.trim() !== "" &&
    expectedTaskId.trim() === selectedTaskId.trim();
}

export const MAIN_INSPECTOR_TABS: Array<{ key: InspectorTabKey; label: string }> = [
  { key: "diff", label: "D" },
  { key: "run", label: "R" },
  { key: "trace", label: "T" },
  { key: "risk", label: "!" }
];

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
    capabilities: []
  });
  const runTaskGraph = ref<OpenAPIContractComponents["schemas"]["AgentGraph"] | null>(null);
  const runTaskGraphLoading = ref(false);
  const runTaskListItems = ref<RunTaskNode[]>([]);
  const runTaskListLoading = ref(false);
  const runTaskListNextCursor = ref<string | null>(null);
  const runTaskStateFilter = ref<RunTaskState | "">("");
  const runTaskConversationId = ref("");
  const runTaskDetail = ref<OpenAPIContractComponents["schemas"]["TaskNode"] | null>(null);
  const runTaskDetailLoading = ref(false);
  const selectedRunTaskId = ref("");
  const composerSuggestions = ref<ComposerSuggestion[]>([]);
  const composerSuggesting = ref(false);
  let composerSuggestSequence = 0;

  const inspectorTabs: Array<{ key: InspectorTabKey; label: string }> = [...MAIN_INSPECTOR_TABS];

  const nonGitCapability: ChangeSetCapability = {
    can_commit: false,
    can_discard: false,
    can_export: true,
    can_export_patch: true,
    reason: "No conversation changeset loaded yet"
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

  const placeholder = computed(() => t("session.placeholderInput"));
  const executionStateCounts = computed(() =>
    runtime.value
      ? getRunStateCounts(runtime.value)
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
        const executionID = event.run_id.trim();
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
  const baseExecutionTraces = computed<RunTraceViewModel[]>(() => {
    const currentRuntime = runtime.value;
    if (!currentRuntime) {
      return [];
    }
    const tracedExecutions = currentRuntime.executions.filter(
      (execution) => execution.agent_config_snapshot?.show_process_trace ?? true
    );
    return buildRunTraceViewModels(currentRuntime.events, tracedExecutions, locale.value);
  });
  const visibleExecutionTraces = computed<RunTraceViewModel[]>(() =>
    baseExecutionTraces.value.filter((trace) => visibleTraceExecutionIds.value.has(trace.executionId))
  );
  const {
    activeTraceCount,
    executionTraces,
    selectedRunTrace,
    selectedTraceMessageId,
    selectedTraceExecutionId,
    selectTraceMessage,
    selectRunTrace
  } = useRunTraceState(visibleExecutionTraces, visibleMessages);
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
    selectRunTrace(executionId);
    actions.openInspectorTab("trace");
    inspectorCollapsed.value = false;
  }

  function showInspectorErrorToast(message: string): void {
    const normalized = message.trim();
    if (normalized === "") {
      return;
    }
    showToast({
      key: INSPECTOR_ERROR_TOAST_KEY,
      tone: "error",
      message: normalized
    });
  }

  function resolveInspectorError(previousError: string, fallback = ""): string {
    const nextError = conversationStore.error.trim();
    if (nextError !== "") {
      return nextError;
    }
    void previousError;
    return fallback.trim();
  }

  function clearRunTaskDisplayState(): void {
    runTaskGraph.value = null;
    runTaskListItems.value = [];
    runTaskListNextCursor.value = null;
    runTaskDetail.value = null;
    runTaskDetailLoading.value = false;
    selectedRunTaskId.value = "";
  }

  function hasRunTaskExecutionContext(): boolean {
    return (runtime.value?.executions ?? []).length > 0;
  }

  function selectTraceExecution(executionId: string): void {
    selectRunTrace(executionId);
  }

  async function refreshRunTaskGraphForActiveConversation(): Promise<boolean> {
    const conversationId = activeConversation.value?.id?.trim() ?? "";
    if (conversationId === "") {
      runTaskGraph.value = null;
      runTaskGraphLoading.value = false;
      return true;
    }
    if (!hasRunTaskExecutionContext()) {
      runTaskGraph.value = null;
      runTaskGraphLoading.value = false;
      return true;
    }
    runTaskGraphLoading.value = true;
    const previousError = conversationStore.error;
    try {
      const graph = await loadConversationRunTaskGraph(conversationId);
      if (!isSameConversationResponseTarget(conversationId, activeConversation.value?.id ?? "")) {
        return false;
      }
      if (graph === null) {
        const errorMessage = resolveInspectorError(previousError);
        if (errorMessage !== "") {
          showInspectorErrorToast(errorMessage);
        }
        return false;
      }
      runTaskGraph.value = graph;
      return true;
    } finally {
      if (isSameConversationResponseTarget(conversationId, activeConversation.value?.id ?? "")) {
        runTaskGraphLoading.value = false;
      }
    }
  }

  async function refreshRunTaskListForActiveConversation(): Promise<boolean> {
    const conversationId = activeConversation.value?.id?.trim() ?? "";
    if (conversationId === "") {
      clearRunTaskState();
      return true;
    }
    if (!hasRunTaskExecutionContext()) {
      clearRunTaskDisplayState();
      runTaskListLoading.value = false;
      return true;
    }
    runTaskListLoading.value = true;
    const previousError = conversationStore.error;
    try {
      const response = await loadConversationRunTasks(conversationId, {
        state: runTaskStateFilter.value === "" ? undefined : runTaskStateFilter.value,
        limit: 20
      });
      if (!isSameConversationResponseTarget(conversationId, activeConversation.value?.id ?? "")) {
        return false;
      }
      if (!response) {
        const errorMessage = resolveInspectorError(previousError);
        if (errorMessage !== "") {
          showInspectorErrorToast(errorMessage);
        }
        return false;
      }
      runTaskListItems.value = response?.items ?? [];
      runTaskListNextCursor.value = response?.next_cursor ?? null;
      const selectedTaskStillExists = runTaskListItems.value.some((task) => task.task_id === selectedRunTaskId.value);
      if (!selectedTaskStillExists) {
        selectedRunTaskId.value = runTaskListItems.value[0]?.task_id ?? "";
      }
      await refreshRunTaskDetailForActiveConversation();
      return true;
    } finally {
      if (isSameConversationResponseTarget(conversationId, activeConversation.value?.id ?? "")) {
        runTaskListLoading.value = false;
      }
    }
  }

  async function loadMoreRunTasksForActiveConversation(): Promise<void> {
    const conversationId = activeConversation.value?.id?.trim() ?? "";
    const cursor = runTaskListNextCursor.value?.trim() ?? "";
    if (conversationId === "" || cursor === "") {
      return;
    }
    runTaskListLoading.value = true;
    const previousError = conversationStore.error;
    try {
      const response = await loadConversationRunTasks(conversationId, {
        state: runTaskStateFilter.value === "" ? undefined : runTaskStateFilter.value,
        cursor,
        limit: 20
      });
      if (!isSameConversationResponseTarget(conversationId, activeConversation.value?.id ?? "")) {
        return;
      }
      if (!response) {
        const errorMessage = resolveInspectorError(previousError);
        if (errorMessage !== "") {
          showInspectorErrorToast(errorMessage);
        }
        return;
      }
      const merged = new Map<string, RunTaskNode>();
      for (const item of runTaskListItems.value) {
        merged.set(item.task_id, item);
      }
      for (const item of response.items) {
        merged.set(item.task_id, item);
      }
      runTaskListItems.value = [...merged.values()];
      runTaskListNextCursor.value = response.next_cursor ?? null;
    } finally {
      if (isSameConversationResponseTarget(conversationId, activeConversation.value?.id ?? "")) {
        runTaskListLoading.value = false;
      }
    }
  }

  async function refreshRunTasksForActiveConversation(): Promise<void> {
    await Promise.all([
      refreshRunTaskGraphForActiveConversation(),
      refreshRunTaskListForActiveConversation()
    ]);
  }

  async function changeRunTaskStateFilterForActiveConversation(state: RunTaskState | ""): Promise<void> {
    const previousState = runTaskStateFilter.value;
    const previousItems = [...runTaskListItems.value];
    const previousCursor = runTaskListNextCursor.value;
    const previousSelectedTaskId = selectedRunTaskId.value;
    const previousTaskDetail = runTaskDetail.value;
    runTaskStateFilter.value = state;
    const refreshed = await refreshRunTaskListForActiveConversation();
    if (!refreshed) {
      runTaskStateFilter.value = previousState;
      runTaskListItems.value = previousItems;
      runTaskListNextCursor.value = previousCursor;
      selectedRunTaskId.value = previousSelectedTaskId;
      runTaskDetail.value = previousTaskDetail;
    }
  }

  async function refreshRunTaskDetailForActiveConversation(): Promise<boolean> {
    const conversationId = activeConversation.value?.id?.trim() ?? "";
    const taskId = selectedRunTaskId.value.trim();
    if (conversationId === "" || taskId === "") {
      runTaskDetail.value = null;
      runTaskDetailLoading.value = false;
      return true;
    }
    if (!hasRunTaskExecutionContext()) {
      runTaskDetail.value = null;
      runTaskDetailLoading.value = false;
      return true;
    }
    runTaskDetailLoading.value = true;
    const previousError = conversationStore.error;
    try {
      const detail = await loadConversationRunTaskById(conversationId, taskId);
      if (!shouldApplyRunTaskDetailResponse(
        conversationId,
        activeConversation.value?.id ?? "",
        taskId,
        selectedRunTaskId.value
      )) {
        return false;
      }
      if (!detail) {
        const errorMessage = resolveInspectorError(previousError);
        if (errorMessage !== "") {
          showInspectorErrorToast(errorMessage);
        }
        return false;
      }
      runTaskDetail.value = detail;
      return true;
    } finally {
      if (shouldApplyRunTaskDetailResponse(
        conversationId,
        activeConversation.value?.id ?? "",
        taskId,
        selectedRunTaskId.value
      )) {
        runTaskDetailLoading.value = false;
      }
    }
  }

  async function selectRunTaskForActiveConversation(taskId: string): Promise<void> {
    const normalizedTaskId = taskId.trim();
    if (normalizedTaskId === "") {
      return;
    }
    const previousTaskId = selectedRunTaskId.value;
    selectedRunTaskId.value = normalizedTaskId;
    const refreshed = await refreshRunTaskDetailForActiveConversation();
    if (!refreshed) {
      selectedRunTaskId.value = previousTaskId;
    }
  }

  async function controlRunTaskForActiveConversation(input: {
    taskId: string;
    action: OpenAPIContractComponents["schemas"]["TaskControlRequest"]["action"];
  }): Promise<void> {
    const conversation = activeConversation.value;
    if (!conversation) {
      return;
    }
    const taskId = input.taskId.trim();
    if (taskId === "") {
      return;
    }
    const previousError = conversationStore.error;
    const controlled = await controlConversationRunTask(conversation, taskId, input.action, `inspector_${input.action}`);
    if (!controlled) {
      const errorMessage = resolveInspectorError(previousError);
      if (errorMessage !== "") {
        showInspectorErrorToast(errorMessage);
      }
      return;
    }
    await refreshRunTasksForActiveConversation();
  }

  function clearRunTaskState(): void {
    runTaskGraph.value = null;
    runTaskGraphLoading.value = false;
    runTaskListItems.value = [];
    runTaskListLoading.value = false;
    runTaskListNextCursor.value = null;
    runTaskStateFilter.value = "";
    runTaskDetail.value = null;
    runTaskDetailLoading.value = false;
    selectedRunTaskId.value = "";
    runTaskConversationId.value = "";
  }

  async function refreshComposerCatalogForActiveConversation(): Promise<void> {
    const conversationId = activeConversation.value?.id?.trim() ?? "";
    if (conversationId === "") {
      composerCatalog.value = {
        revision: "",
        commands: [],
        capabilities: []
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
        capabilities: []
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
    return catalog.capabilities
      .filter((resource) => resource.kind === resourceType)
      .map((resource) => ({
        ...resource,
        mentionId: resolveCapabilityMentionId(resource.id)
      }))
      .filter((resource) => resource.mentionId !== "")
      .filter((resource) => {
        if (resourceQuery === "") {
          return true;
        }
        return resource.mentionId.toLowerCase().includes(resourceQuery) || resource.name.toLowerCase().includes(resourceQuery);
      })
      .slice(0, limit)
      .map((resource) => ({
        kind: "resource",
        label: `@${resource.kind}:${resource.mentionId}`,
        detail: resolveResourceSuggestionDetail(resource),
        insert_text: `@${resource.kind}:${resource.mentionId}`,
        replace_start: tokenStart,
        replace_end: tokenEnd
      }));
  }

  function resolveResourceTypeSuggestionDetail(type: string): string {
    switch (type) {
      case "model":
        return t("session.composer.suggestion.type.model");
      case "rule":
        return t("session.composer.suggestion.type.rule");
      case "skill":
        return t("session.composer.suggestion.type.skill");
      case "mcp":
        return t("session.composer.suggestion.type.mcp");
      default:
        return "";
    }
  }

  function resolveResourceSuggestionDetail(resource: { kind: string; id: string; mentionId: string; name: string; description?: string }): string {
    if (resource.kind === "file") {
      return "";
    }
    const normalizedName = stringsOrEmpty(resource.name);
    const normalizedID = stringsOrEmpty(resource.mentionId);
    if (normalizedName === "" || normalizedName.toLowerCase() === normalizedID.toLowerCase()) {
      return stringsOrEmpty(resource.description);
    }
    return normalizedName;
  }

  function resolveCapabilityMentionId(capabilityId: string): string {
    const normalized = stringsOrEmpty(capabilityId);
    const separatorIndex = normalized.indexOf(":");
    if (separatorIndex < 0) {
      return normalized;
    }
    return normalized.slice(separatorIndex + 1);
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
    () => ({
      conversationID: activeConversation.value?.id ?? "",
      conversationModelID: activeConversation.value?.model_config_id ?? ""
    }),
    ({ conversationID, conversationModelID }) => {
      const runtimeModelID = runtime.value?.modelId ?? "";
      if (conversationID === "") {
        return;
      }
      if (conversationModelID.trim() === "" || conversationModelID.trim() === runtimeModelID.trim()) {
        return;
      }
      setConversationModel(conversationID, conversationModelID);
    },
    { immediate: true }
  );

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
      void refreshConversationChangeSet(nextId);
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
    () => ({
      conversationId: activeConversation.value?.id ?? "",
      hydrated: runtime.value?.hydrated ?? false,
      changeSetID: runtime.value?.changeSet?.change_set_id ?? ""
    }),
    ({ conversationId, hydrated, changeSetID }) => {
      if (conversationId === "" || !hydrated || changeSetID !== "") {
        return;
      }
      void refreshConversationChangeSet(conversationId);
    },
    { immediate: true }
  );

  watch(
    () => workspaceStore.currentWorkspaceId,
    () => {
      streamCoordinator.clearStreams();
      projectImportInProgress.value = false;
      projectImportFeedback.value = "";
      projectImportError.value = "";
      clearRunTaskState();
      void refreshResourceConfigsByType("model");
      void refreshComposerCatalogForActiveConversation();
      streamCoordinator.syncConversationStreams();
    }
  );
  watch(
    () => ({
      conversationId: activeConversation.value?.id ?? "",
      inspectorTab: runtime.value?.inspectorTab ?? "diff",
      executionFingerprint: (runtime.value?.executions ?? [])
        .map((item) => `${item.id}:${item.state}:${item.updated_at}`)
        .join("|")
    }),
    ({ conversationId, inspectorTab }) => {
      if (conversationId === "") {
        clearRunTaskState();
        return;
      }
      if (runTaskConversationId.value !== conversationId) {
        runTaskConversationId.value = conversationId;
        runTaskListItems.value = [];
        runTaskListNextCursor.value = null;
        runTaskStateFilter.value = "";
        runTaskDetail.value = null;
        runTaskDetailLoading.value = false;
        selectedRunTaskId.value = "";
      }
      if (inspectorTab !== "run") {
        return;
      }
      void refreshRunTasksForActiveConversation();
    },
    { immediate: true }
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
    refreshRunTaskGraph: refreshRunTasksForActiveConversation,
    changeRunTaskStateFilter: changeRunTaskStateFilterForActiveConversation,
    selectRunTask: selectRunTaskForActiveConversation,
    loadMoreRunTasks: loadMoreRunTasksForActiveConversation,
    controlRunTask: controlRunTaskForActiveConversation,
    clearComposerSuggestions,
    executionTraces,
    runTaskGraph,
    runTaskGraphLoading,
    runTaskListItems,
    runTaskListLoading,
    runTaskListNextCursor,
    runTaskStateFilter,
    selectedRunTask: runTaskDetail,
    runTaskDetailLoading,
    selectedRunTrace,
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
    selectTraceExecution,
    selectTraceMessage,
    selectTraceInInspector,
    selectedTraceMessageId,
    workspaceLabel,
    workspaceStore
  };
}

function toPendingQuestionFromEvent(
  event: RunLifecycleEvent,
  executionByID: Map<string, { id: string; message_id: string; queue_index: number }>
): PendingExecutionQuestionViewModel | null {
  if (event.type !== "thinking_delta") {
    return null;
  }
  if (stringOrEmpty(event.payload.stage) !== "run_user_question_needed") {
    return null;
  }
  const executionID = event.run_id.trim();
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

function isPendingQuestionResolvedEvent(event: RunLifecycleEvent): boolean {
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
