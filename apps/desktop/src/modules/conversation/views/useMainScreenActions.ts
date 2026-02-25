import type { ComputedRef, Ref } from "vue";
import type { Router } from "vue-router";
import {
  commitLatestDiff,
  discardLatestDiff,
  getLatestFinishedExecution,
  rollbackConversationToMessage,
  setConversationDraft,
  setConversationError,
  setConversationInspectorTab,
  setConversationMode,
  setConversationModel,
  stopConversationExecution,
  submitConversationMessage
} from "@/modules/conversation/store";
import { exportExecutionPatch } from "@/modules/conversation/services";
import type { ConversationRuntime } from "@/modules/conversation/store/state";
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
import { workspaceStore } from "@/shared/stores/workspaceStore";
import type { Conversation, InspectorTabKey, Project } from "@/shared/types/api";
import { setWorkspaceConnection, switchWorkspaceContext, upsertWorkspace } from "@/modules/workspace/store";
type MainScreenActionsInput = {
  router: Router;
  activeConversation: ComputedRef<Conversation | undefined>;
  activeProject: ComputedRef<Project | undefined>;
  runtime: ComputedRef<ConversationRuntime | undefined>;
  modelOptions: ComputedRef<Array<{ value: string; label: string }>>;
  inspectorCollapsed: Ref<boolean>;
  editingConversationName: Ref<boolean>;
  conversationNameDraft: Ref<string>;
  projectImportInProgress: Ref<boolean>;
  projectImportFeedback: Ref<string>;
  projectImportError: Ref<string>;
  resolveSemanticModelID: (raw: string) => string;
};
export function useMainScreenActions(input: MainScreenActionsInput) {
  function updateDraft(value: string): void {
    if (!input.activeConversation.value) {
      return;
    }
    setConversationDraft(input.activeConversation.value.id, value);
  }
  async function updateMode(value: "agent" | "plan"): Promise<void> {
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
    const previousModel = input.resolveSemanticModelID(input.runtime.value?.modelId ?? input.activeConversation.value.model_id);
    setConversationModel(conversationId, targetModelID);
    const updated = await updateConversationModelById(projectId, conversationId, targetModelID);
    if (!updated) {
      setConversationModel(conversationId, previousModel);
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
  async function sendMessage(): Promise<void> {
    if (!input.activeConversation.value || !input.activeProject.value) {
      return;
    }
    await submitConversationMessage(input.activeConversation.value, input.activeProject.value.is_git);
  }
  async function stopExecution(): Promise<void> {
    if (!input.activeConversation.value) {
      return;
    }
    await stopConversationExecution(input.activeConversation.value);
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
  async function commitDiff(): Promise<void> {
    if (!input.activeConversation.value) {
      return;
    }
    await commitLatestDiff(input.activeConversation.value.id);
  }
  async function discardDiff(): Promise<void> {
    if (!input.activeConversation.value) {
      return;
    }
    await discardLatestDiff(input.activeConversation.value.id);
  }
  async function exportPatch(): Promise<void> {
    if (!input.activeConversation.value) {
      return;
    }
    const execution = getLatestFinishedExecution(input.activeConversation.value.id);
    if (!execution) {
      return;
    }
    try {
      const patch = await exportExecutionPatch(execution.id);
      const blob = new Blob([patch], { type: "text/plain;charset=utf-8" });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `${execution.id}.patch`;
      link.click();
      URL.revokeObjectURL(url);
    } catch (error) {
      setConversationError(toDisplayError(error));
    }
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
    sendMessage,
    loginWorkspace,
    startEditConversationName,
    stopExecution,
    switchWorkspace,
    updateDraft,
    updateMode,
    updateModel
  };
}
