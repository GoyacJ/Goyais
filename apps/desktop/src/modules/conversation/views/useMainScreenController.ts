import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";

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
  loadNextConversationsPage,
  loadNextProjectsPage,
  loadPreviousConversationsPage,
  loadPreviousProjectsPage,
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
import type { DiffCapability, InspectorTabKey } from "@/shared/types/api";

export function useMainScreenController() {
  const router = useRouter();
  const { t } = useI18n();

  const editingConversationName = ref(false);
  const conversationNameDraft = ref("");
  const inspectorCollapsed = ref(false);
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

  watch(
    () => workspaceStore.currentWorkspaceId,
    () => {
      projectImportInProgress.value = false;
      projectImportFeedback.value = "";
      projectImportError.value = "";
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

  function openInspectorTab(tab: InspectorTabKey): void {
    inspectorCollapsed.value = false;
    changeInspectorTab(tab);
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

  async function importProjectDirectoryAction(repoPath: string): Promise<void> {
    const normalizedPath = repoPath.trim();
    if (normalizedPath === "") {
      return;
    }

    projectImportInProgress.value = true;
    projectImportFeedback.value = "";
    projectImportError.value = "";

    try {
      const created = await importProjectByDirectory(normalizedPath);
      if (!created) {
        projectImportError.value = projectStore.error || "PROJECT_IMPORT_FAILED: 导入项目失败";
        return;
      }
      projectImportFeedback.value = `已添加项目：${created.name}`;
      projectImportError.value = "";
    } finally {
      projectImportInProgress.value = false;
    }
  }

  async function deleteProjectById(projectId: string): Promise<void> {
    await deleteProject(projectId);
  }

  async function addConversationByPrompt(projectId: string): Promise<void> {
    const project = projectStore.projects.find((item) => item.id === projectId);
    if (!project) {
      return;
    }
    const nextIndex = (projectStore.conversationsByProjectId[project.id] ?? []).length + 1;
    await addConversation(project, `新对话 ${nextIndex}`);
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
  }

  async function switchWorkspace(workspaceId: string): Promise<void> {
    await switchWorkspaceContext(workspaceId);
  }

  async function paginateProjects(direction: "prev" | "next"): Promise<void> {
    if (direction === "next") {
      await loadNextProjectsPage();
      return;
    }
    await loadPreviousProjectsPage();
  }

  async function paginateConversations(projectId: string, direction: "prev" | "next"): Promise<void> {
    if (direction === "next") {
      await loadNextConversationsPage(projectId);
      return;
    }
    await loadPreviousConversationsPage(projectId);
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

  return {
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
    importProjectDirectory: importProjectDirectoryAction,
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
  };
}
