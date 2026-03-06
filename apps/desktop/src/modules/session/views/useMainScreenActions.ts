import { computed, ref, type ComputedRef, type Ref } from "vue";
import type { Router } from "vue-router";
import {
  answerConversationExecutionQuestion,
  approveConversationExecution,
  commitConversationChangeset,
  conversationStore,
  denyConversationExecution,
  discardConversationChangeset,
  removeQueuedConversationExecution,
  rollbackConversationToMessage,
  setConversationDraft,
  setConversationError,
  setConversationInspectorTab,
  setConversationMode,
  setConversationModel,
  stopConversationExecution,
  submitConversationMessage
} from "@/modules/session/store";
import { exportConversationChangeSet } from "@/modules/session/services";
import type { SessionRuntime } from "@/modules/session/store/state";
import { buildNameFromFirstMessage, isDefaultConversationName } from "@/modules/session/views/conversationNamePolicy";
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
  renameConversationById,
  setActiveConversation,
  setActiveProject,
  updateConversationModeById,
  updateConversationModelById
} from "@/modules/project/store";
import { refreshAdminData } from "@/modules/admin/store";
import { refreshResources, refreshModelCatalog } from "@/modules/resource/store";
import { refreshProjects } from "@/modules/project/store";
import { createRemoteConnection, loginWorkspace as loginWorkspaceRequest } from "@/modules/workspace/services";
import { refreshMeForCurrentWorkspace, setWorkspaceToken } from "@/shared/stores/authStore";
import { toDisplayError } from "@/shared/services/errorMapper";
import { showToast } from "@/shared/stores/toastStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import type { InspectorTabKey, PermissionMode, Project, Session } from "@/shared/types/api";
import { setWorkspaceConnection, switchWorkspaceContext, upsertWorkspace } from "@/modules/workspace/store";

const INSPECTOR_ERROR_TOAST_KEY = "session-inspector-error";

type MainScreenActionsInput = {
  router: Router;
  activeConversation: ComputedRef<Session | undefined>;
  activeProject: ComputedRef<Project | undefined>;
  runtime: ComputedRef<SessionRuntime | undefined>;
  modelOptions: ComputedRef<Array<{ value: string; label: string }>>;
  composerCatalogRevision: ComputedRef<string>;
  inspectorCollapsed: Ref<boolean>;
  editingConversationName: Ref<boolean>;
  conversationNameDraft: Ref<string>;
  projectImportInProgress: Ref<boolean>;
  projectImportFeedback: Ref<string>;
  projectImportError: Ref<string>;
  resolveSemanticModelID: (raw: string) => string;
};
export function useMainScreenActions(input: MainScreenActionsInput) {
  const modelSwitchingConversationID = ref("");
  const modelSwitchPromiseByConversationID = new Map<string, Promise<boolean>>();

  const isSwitchingModel = computed<boolean>(() => {
    const conversationID = input.activeConversation.value?.id?.trim() ?? "";
    if (conversationID === "") {
      return false;
    }
    return modelSwitchingConversationID.value === conversationID;
  });

  function updateDraft(value: string): void {
    if (!input.activeConversation.value) {
      return;
    }
    setConversationDraft(input.activeConversation.value.id, value);
  }
  async function updateMode(value: PermissionMode): Promise<void> {
    if (!input.activeConversation.value || !input.activeProject.value) {
      return;
    }
    const conversationId = input.activeConversation.value.id;
    const projectId = input.activeProject.value.id;
    const previousMode = input.runtime.value?.mode ?? input.activeConversation.value.default_mode;
    setConversationMode(conversationId, value);
    const updated = await updateConversationModeById(projectId, conversationId, value);
    if (!updated) {
      setConversationMode(conversationId, previousMode);
    }
  }
  async function updateModel(value: string): Promise<void> {
    if (!input.activeConversation.value || !input.activeProject.value) {
      return;
    }
    const targetModelID = input.resolveSemanticModelID(value);
    if (targetModelID === "") {
      return;
    }
    const conversationId = input.activeConversation.value.id;
    const projectId = input.activeProject.value.id;
    const previousModel = input.resolveSemanticModelID(
      input.runtime.value?.modelId ?? input.activeConversation.value.model_config_id
    );
    setConversationModel(conversationId, targetModelID);
    modelSwitchingConversationID.value = conversationId;
    const pending = (async () => {
      const updated = await updateConversationModelById(projectId, conversationId, targetModelID);
      if (!updated) {
        setConversationModel(conversationId, previousModel);
      }
      return updated;
    })();
    modelSwitchPromiseByConversationID.set(conversationId, pending);
    try {
      await pending;
    } finally {
      const currentPending = modelSwitchPromiseByConversationID.get(conversationId);
      if (currentPending === pending) {
        modelSwitchPromiseByConversationID.delete(conversationId);
      }
      if (modelSwitchingConversationID.value === conversationId) {
        modelSwitchingConversationID.value = "";
      }
    }
  }
  function changeInspectorTab(tab: InspectorTabKey): void {
    if (!input.activeConversation.value) {
      return;
    }
    setConversationInspectorTab(input.activeConversation.value.id, tab);
  }
  function openInspectorTab(tab: InspectorTabKey): void {
    input.inspectorCollapsed.value = false;
    changeInspectorTab(tab);
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

  function resolveConversationError(previousError: string, fallback: string): string {
    const nextError = conversationStore.error.trim();
    if (nextError !== "") {
      return nextError;
    }
    void previousError;
    return fallback.trim();
  }

  async function sendMessage(): Promise<void> {
    if (!input.activeConversation.value || !input.activeProject.value) {
      return;
    }
    const conversation = input.activeConversation.value;
    const project = input.activeProject.value;
    const runtime = input.runtime.value;
    const selectedModelID = input.resolveSemanticModelID(
      runtime?.modelId ?? conversation.model_config_id
    );
    const hasAllowedModel = selectedModelID !== "" && input.modelOptions.value.some((item) => item.value === selectedModelID);
    if (!hasAllowedModel) {
      setConversationError("当前项目未绑定可用模型，请先在项目配置中绑定模型");
      return;
    }
    const switchingPromise = modelSwitchPromiseByConversationID.get(conversation.id);
    if (switchingPromise) {
      const switched = await switchingPromise;
      if (!switched) {
        setConversationError("模型切换失败，请重试后再发送");
        return;
      }
    }

    const conversationId = conversation.id;
    const projectId = project.id;
    const conversationName = conversation.name;
    const firstDraft = runtime?.draft ?? "";
    const hasUserMessageBeforeSend = (runtime?.messages ?? []).some((message) => message.role === "user");
    const nextName = buildNameFromFirstMessage(firstDraft);
    const shouldAutoRename =
      isDefaultConversationName(conversationName) && !hasUserMessageBeforeSend && nextName !== "";

    try {
      await submitConversationMessage(conversation, project.is_git, {
        catalogRevision: input.composerCatalogRevision.value
      });
    } catch (error) {
      setConversationError(toDisplayError(error));
    }

    if (shouldAutoRename) {
      await renameConversationById(projectId, conversationId, nextName);
    }
  }
  async function stopExecution(): Promise<void> {
    if (!input.activeConversation.value) {
      return;
    }
    await stopConversationExecution(input.activeConversation.value);
  }
  async function removeQueuedMessage(executionID: string): Promise<void> {
    if (!input.activeConversation.value) {
      return;
    }
    await removeQueuedConversationExecution(input.activeConversation.value, executionID);
  }
  async function approveExecution(): Promise<void> {
    if (!input.activeConversation.value) {
      return;
    }
    await approveConversationExecution(input.activeConversation.value);
  }
  async function denyExecution(): Promise<void> {
    if (!input.activeConversation.value) {
      return;
    }
    await denyConversationExecution(input.activeConversation.value);
  }
  async function answerExecutionQuestion(inputPayload: {
    executionId: string;
    questionId: string;
    selectedOptionId?: string;
    text?: string;
  }): Promise<void> {
    if (!input.activeConversation.value) {
      return;
    }
    await answerConversationExecutionQuestion(input.activeConversation.value, inputPayload);
  }
  async function rollbackMessage(messageId: string): Promise<void> {
    if (!input.activeConversation.value) {
      return;
    }
    await rollbackConversationToMessage(input.activeConversation.value.id, messageId);
  }
  async function importProjectDirectoryAction(repoPath: string): Promise<void> {
    const normalizedPath = repoPath.trim();
    if (normalizedPath === "") {
      return;
    }
    input.projectImportInProgress.value = true;
    input.projectImportFeedback.value = "";
    input.projectImportError.value = "";
    try {
      const created = await importProjectByDirectory(normalizedPath);
      if (!created) {
        input.projectImportError.value = projectStore.error || "PROJECT_IMPORT_FAILED: 导入项目失败";
        return;
      }
      input.projectImportFeedback.value = `已添加项目：${created.name}`;
      input.projectImportError.value = "";
    } finally {
      input.projectImportInProgress.value = false;
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
    await addConversation(project, `新会话 ${nextIndex}`);
  }
  async function deleteConversationById(projectId: string, conversationId: string): Promise<void> {
    await deleteConversation(projectId, conversationId);
  }
  function selectConversation(projectId: string, conversationId: string): void {
    setActiveProject(projectId);
    setActiveConversation(conversationId);
  }
  async function renameConversation(projectId: string, conversationId: string, name: string): Promise<void> {
    const normalizedName = name.trim();
    if (projectId.trim() === "" || conversationId.trim() === "" || normalizedName === "") {
      return;
    }
    await renameConversationById(projectId, conversationId, normalizedName);
  }
  async function exportConversation(conversationId: string): Promise<void> {
    const markdown = await exportConversationById(conversationId);
    if (!markdown) {
      return;
    }
    const blob = new Blob([markdown], { type: "text/markdown;charset=utf-8" });
    triggerBlobDownload(blob, `${conversationId}.md`);
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
  async function loginWorkspace(payload: { workspaceId: string; username?: string; password?: string; token?: string }): Promise<void> {
    const workspaceId = payload.workspaceId.trim();
    if (workspaceId === "") {
      return;
    }

    try {
      const response = await loginWorkspaceRequest({
        workspace_id: workspaceId,
        username: payload.username?.trim() || undefined,
        password: payload.password || undefined,
        token: payload.token?.trim() || undefined
      });

      setWorkspaceToken(workspaceId, response.access_token, response.refresh_token);

      const currentWorkspaceId = workspaceStore.currentWorkspaceId.trim();
      if (currentWorkspaceId === workspaceId) {
        await refreshMeForCurrentWorkspace();
      } else {
        await switchWorkspaceContext(workspaceId);
      }

      if (workspaceStore.connectionState === "ready") {
        await Promise.all([refreshProjects(), refreshResources(), refreshModelCatalog()]);
        if (workspaceStore.mode === "remote") {
          await refreshAdminData();
        }
      }
    } catch (error) {
      setConversationError(toDisplayError(error));
    }
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
    void input.router.push("/remote/account");
  }
  function openSettings(): void {
    void input.router.push("/settings/theme");
  }
  function startEditConversationName(): void {
    if (!input.activeConversation.value) {
      return;
    }
    input.conversationNameDraft.value = input.activeConversation.value.name;
    input.editingConversationName.value = true;
  }
  function onConversationNameInput(event: Event): void {
    input.conversationNameDraft.value = (event.target as HTMLInputElement).value;
  }
  async function saveConversationName(): Promise<void> {
    if (!input.editingConversationName.value || !input.activeConversation.value || !input.activeProject.value) {
      input.editingConversationName.value = false;
      return;
    }
    const name = input.conversationNameDraft.value.trim();
    input.editingConversationName.value = false;
    if (name === "" || name === input.activeConversation.value.name) {
      return;
    }
    await renameConversationById(input.activeProject.value.id, input.activeConversation.value.id, name);
  }
  async function commitDiff(message = ""): Promise<boolean> {
    if (!input.activeConversation.value) {
      return false;
    }
    const previousError = conversationStore.error;
    const committed = await commitConversationChangeset(input.activeConversation.value.id, message);
    if (!committed) {
      showInspectorErrorToast(resolveConversationError(previousError, "CHANGESET_COMMIT_FAILED"));
    }
    return committed;
  }
  async function discardDiff(): Promise<boolean> {
    if (!input.activeConversation.value) {
      return false;
    }
    const previousError = conversationStore.error;
    const discarded = await discardConversationChangeset(input.activeConversation.value.id);
    if (!discarded) {
      showInspectorErrorToast(resolveConversationError(previousError, "CHANGESET_DISCARD_FAILED"));
    }
    return discarded;
  }
  async function exportPatch(): Promise<boolean> {
    const conversationId = input.activeConversation.value?.id?.trim() ?? "";
    if (conversationId === "") {
      const message = "CONVERSATION_NOT_FOUND: no active conversation";
      setConversationError(message);
      showInspectorErrorToast(message);
      return false;
    }
    const canExport = input.runtime.value?.changeSet?.capability.can_export ?? true;
    if (!canExport) {
      const message = input.runtime.value?.changeSet?.capability.reason ?? "CHANGESET_EXPORT_DISABLED";
      setConversationError(message);
      showInspectorErrorToast(message);
      return false;
    }
    try {
      const filesArchive = await exportConversationChangeSet(conversationId);
      const bytesBuffer = decodeBase64(filesArchive.archive_base64);
      const blob = new Blob([bytesBuffer], { type: "application/zip" });
      triggerBlobDownload(blob, filesArchive.file_name.trim() || `${conversationId}-changeset.zip`);
      return true;
    } catch (error) {
      const message = toDisplayError(error);
      setConversationError(message);
      showInspectorErrorToast(message);
      return false;
    }
  }

  function decodeBase64(inputBase64: string): ArrayBuffer {
    const normalized = inputBase64.trim();
    if (normalized === "") {
      return new ArrayBuffer(0);
    }
    const binary = atob(normalized);
    const bytes = new Uint8Array(binary.length);
    for (let index = 0; index < binary.length; index += 1) {
      bytes[index] = binary.charCodeAt(index);
    }
    const buffer = new ArrayBuffer(bytes.length);
    new Uint8Array(buffer).set(bytes);
    return buffer;
  }
  function triggerBlobDownload(blob: Blob, fileName: string): void {
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = fileName;
    link.style.display = "none";
    document.body.append(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  }
  return {
    addConversationByPrompt,
    changeInspectorTab,
    commitDiff,
    createWorkspace,
    deleteConversationById,
    deleteProjectById,
    discardDiff,
    exportConversation,
    exportPatch,
    importProjectDirectory: importProjectDirectoryAction,
    onConversationNameInput,
    openAccount,
    openInspectorTab,
    openSettings,
    paginateConversations,
    paginateProjects,
    rollbackMessage,
    saveConversationName,
    selectConversation,
    renameConversation,
    sendMessage,
    removeQueuedMessage,
    approveExecution,
    denyExecution,
    answerExecutionQuestion,
    loginWorkspace,
    startEditConversationName,
    stopExecution,
    switchWorkspace,
    isSwitchingModel,
    updateDraft,
    updateMode,
    updateModel
  };
}
