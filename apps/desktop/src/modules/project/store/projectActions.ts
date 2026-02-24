import {
  createConversation,
  createProject,
  exportConversationMarkdown,
  importProjectDirectory,
  patchConversation,
  removeConversation,
  removeProject,
  renameConversation,
  updateProjectConfig
} from "@/modules/project/services";
import {
  refreshConversationsForActiveProject,
  refreshProjects
} from "@/modules/project/store/paginationActions";
import { projectStore, resolveCurrentWorkspaceToken, resolveWorkspaceToken } from "@/modules/project/store/state";
import { toDisplayError } from "@/shared/services/errorMapper";
import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";
import type { Conversation, Project, ProjectConfig } from "@/shared/types/api";

export async function addProject(input: { name: string; repo_path: string; is_git: boolean }): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }
  const token = resolveWorkspaceToken(workspace.id);
  projectStore.error = "";

  try {
    const created = await createProject(workspace.id, input, { token });
    projectStore.activeProjectId = created.id;
    projectStore.activeConversationId = "";
    await refreshProjects();
  } catch (error) {
    projectStore.error = toDisplayError(error);
  }
}

export async function importProjectByDirectory(repoPath: string): Promise<Project | null> {
  const workspace = getCurrentWorkspace();
  if (!workspace || repoPath.trim() === "") {
    return null;
  }
  const token = resolveWorkspaceToken(workspace.id);
  projectStore.error = "";

  try {
    const created = await importProjectDirectory(workspace.id, repoPath, { token });
    projectStore.activeProjectId = created.id;
    projectStore.activeConversationId = "";
    await refreshProjects();
    projectStore.error = "";
    return created;
  } catch (error) {
    projectStore.error = toDisplayError(error);
    return null;
  }
}

export async function deleteProject(projectId: string): Promise<void> {
  const token = resolveCurrentWorkspaceToken();
  try {
    await removeProject(projectId, { token });
    projectStore.projects = projectStore.projects.filter((project) => project.id !== projectId);
    delete projectStore.conversationsByProjectId[projectId];
    delete projectStore.conversationPagesByProjectId[projectId];
    delete projectStore.projectConfigsByProjectId[projectId];

    if (projectStore.activeProjectId === projectId) {
      projectStore.activeProjectId = projectStore.projects[0]?.id ?? "";
      projectStore.activeConversationId = "";
      await refreshConversationsForActiveProject();
    }
  } catch (error) {
    projectStore.error = toDisplayError(error);
  }
}

export async function addConversation(project: Project, name: string): Promise<Conversation | null> {
  const token = resolveWorkspaceToken(project.workspace_id);
  try {
    const created = await createConversation(project, name, { token });
    await refreshConversationsForActiveProject();
    projectStore.activeConversationId = created.id;
    return created;
  } catch (error) {
    projectStore.error = toDisplayError(error);
    return null;
  }
}

export async function renameConversationById(projectId: string, conversationId: string, name: string): Promise<void> {
  const token = resolveCurrentWorkspaceToken();
  try {
    const updated = await renameConversation(conversationId, name, { token });
    const list = projectStore.conversationsByProjectId[projectId] ?? [];
    projectStore.conversationsByProjectId[projectId] = list.map((conversation) =>
      conversation.id === conversationId ? updated : conversation
    );
  } catch (error) {
    projectStore.error = toDisplayError(error);
  }
}

export async function updateConversationModeById(
  projectId: string,
  conversationId: string,
  mode: Conversation["default_mode"]
): Promise<boolean> {
  const token = resolveCurrentWorkspaceToken();
  try {
    const updated = await patchConversation(conversationId, { mode }, { token });
    const list = projectStore.conversationsByProjectId[projectId] ?? [];
    projectStore.conversationsByProjectId[projectId] = list.map((conversation) =>
      conversation.id === conversationId ? updated : conversation
    );
    return true;
  } catch (error) {
    projectStore.error = toDisplayError(error);
    return false;
  }
}

export async function updateConversationModelById(
  projectId: string,
  conversationId: string,
  modelId: string
): Promise<boolean> {
  const token = resolveCurrentWorkspaceToken();
  try {
    const updated = await patchConversation(conversationId, { model_id: modelId }, { token });
    const list = projectStore.conversationsByProjectId[projectId] ?? [];
    projectStore.conversationsByProjectId[projectId] = list.map((conversation) =>
      conversation.id === conversationId ? updated : conversation
    );
    return true;
  } catch (error) {
    projectStore.error = toDisplayError(error);
    return false;
  }
}

export async function deleteConversation(projectId: string, conversationId: string): Promise<void> {
  void projectId;
  const token = resolveCurrentWorkspaceToken();
  try {
    await removeConversation(conversationId, { token });
    await refreshConversationsForActiveProject();
    if (projectStore.activeConversationId === conversationId || projectStore.activeConversationId === "") {
      projectStore.activeConversationId = "";
    }
  } catch (error) {
    projectStore.error = toDisplayError(error);
  }
}

export async function exportConversationById(conversationId: string): Promise<string | null> {
  const token = resolveCurrentWorkspaceToken();
  try {
    return await exportConversationMarkdown(conversationId, { token });
  } catch (error) {
    projectStore.error = toDisplayError(error);
    return null;
  }
}

export async function updateProjectBinding(
  projectId: string,
  config: Omit<ProjectConfig, "project_id" | "updated_at">
): Promise<void> {
  const token = resolveCurrentWorkspaceToken();
  try {
    const updated = await updateProjectConfig(projectId, config, { token });
    projectStore.projectConfigsByProjectId[projectId] = updated;
  } catch (error) {
    projectStore.error = toDisplayError(error);
  }
}

export function setActiveProject(projectId: string): void {
  projectStore.activeProjectId = projectId;
  projectStore.activeConversationId = "";
}

export function setActiveConversation(conversationId: string): void {
  projectStore.activeConversationId = conversationId;
}
