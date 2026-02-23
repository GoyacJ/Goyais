import { reactive } from "vue";

import {
  createConversation,
  createProject,
  exportConversationMarkdown,
  importProjectDirectory,
  listConversations,
  listProjects,
  removeConversation,
  removeProject,
  renameConversation,
  updateProjectConfig
} from "@/modules/project/services";
import { toDisplayError } from "@/shared/services/errorMapper";
import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";
import type { Conversation, Project, ProjectConfig } from "@/shared/types/api";

type ProjectState = {
  projects: Project[];
  conversationsByProjectId: Record<string, Conversation[]>;
  projectConfigsByProjectId: Record<string, ProjectConfig>;
  activeProjectId: string;
  activeConversationId: string;
  loading: boolean;
  error: string;
};

const initialState: ProjectState = {
  projects: [],
  conversationsByProjectId: {},
  projectConfigsByProjectId: {},
  activeProjectId: "",
  activeConversationId: "",
  loading: false,
  error: ""
};

export const projectStore = reactive<ProjectState>({ ...initialState });

export function resetProjectStore(): void {
  projectStore.projects = [];
  projectStore.conversationsByProjectId = {};
  projectStore.projectConfigsByProjectId = {};
  projectStore.activeProjectId = "";
  projectStore.activeConversationId = "";
  projectStore.loading = false;
  projectStore.error = "";
}

export async function refreshProjects(): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  projectStore.loading = true;
  projectStore.error = "";

  try {
    const response = await listProjects(workspace.id);
    projectStore.projects = response.items;
    projectStore.activeProjectId = projectStore.activeProjectId || projectStore.projects[0]?.id || "";
    await refreshConversationsForActiveProject();
  } catch (error) {
    projectStore.error = toDisplayError(error);
  } finally {
    projectStore.loading = false;
  }
}

export async function refreshConversationsForActiveProject(): Promise<void> {
  const projectId = projectStore.activeProjectId;
  if (projectId === "") {
    return;
  }

  try {
    const response = await listConversations(projectId);
    projectStore.conversationsByProjectId[projectId] = response.items;
    projectStore.activeConversationId = projectStore.activeConversationId || response.items[0]?.id || "";
  } catch (error) {
    projectStore.error = toDisplayError(error);
  }
}

export async function addProject(input: { name: string; repo_path: string; is_git: boolean }): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  try {
    const created = await createProject(workspace.id, input);
    projectStore.projects.push(created);
    projectStore.activeProjectId = created.id;
    projectStore.conversationsByProjectId[created.id] = [];
  } catch (error) {
    projectStore.error = toDisplayError(error);
  }
}

export async function importProjectByDirectory(repoPath: string): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace || repoPath.trim() === "") {
    return;
  }

  try {
    const created = await importProjectDirectory(workspace.id, repoPath);
    projectStore.projects.push(created);
    projectStore.activeProjectId = created.id;
    projectStore.conversationsByProjectId[created.id] = [];
  } catch (error) {
    projectStore.error = toDisplayError(error);
  }
}

export async function deleteProject(projectId: string): Promise<void> {
  try {
    await removeProject(projectId);
    projectStore.projects = projectStore.projects.filter((project) => project.id !== projectId);
    delete projectStore.conversationsByProjectId[projectId];
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
  try {
    const created = await createConversation(project, name);
    const list = projectStore.conversationsByProjectId[project.id] ?? [];
    projectStore.conversationsByProjectId[project.id] = [...list, created];
    projectStore.activeConversationId = created.id;
    return created;
  } catch (error) {
    projectStore.error = toDisplayError(error);
    return null;
  }
}

export async function renameConversationById(projectId: string, conversationId: string, name: string): Promise<void> {
  try {
    const updated = await renameConversation(conversationId, name);
    const list = projectStore.conversationsByProjectId[projectId] ?? [];
    projectStore.conversationsByProjectId[projectId] = list.map((conversation) =>
      conversation.id === conversationId ? updated : conversation
    );
  } catch (error) {
    projectStore.error = toDisplayError(error);
  }
}

export async function deleteConversation(projectId: string, conversationId: string): Promise<void> {
  try {
    await removeConversation(conversationId);
    const list = projectStore.conversationsByProjectId[projectId] ?? [];
    projectStore.conversationsByProjectId[projectId] = list.filter((conversation) => conversation.id !== conversationId);
    if (projectStore.activeConversationId === conversationId) {
      projectStore.activeConversationId = projectStore.conversationsByProjectId[projectId]?.[0]?.id ?? "";
    }
  } catch (error) {
    projectStore.error = toDisplayError(error);
  }
}

export async function exportConversationById(conversationId: string): Promise<string | null> {
  try {
    return await exportConversationMarkdown(conversationId);
  } catch (error) {
    projectStore.error = toDisplayError(error);
    return null;
  }
}

export async function updateProjectBinding(
  projectId: string,
  config: Omit<ProjectConfig, "project_id" | "updated_at">
): Promise<void> {
  try {
    const updated = await updateProjectConfig(projectId, config);
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
