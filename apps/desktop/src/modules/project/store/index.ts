import { reactive } from "vue";

import {
  createConversation,
  createProject,
  exportConversationMarkdown,
  getProjectConfig,
  importProjectDirectory,
  listConversations,
  listProjects,
  listWorkspaceProjectConfigs,
  removeConversation,
  removeProject,
  renameConversation,
  updateProjectConfig
} from "@/modules/project/services";
import { toDisplayError } from "@/shared/services/errorMapper";
import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";
import type { Conversation, Project, ProjectConfig } from "@/shared/types/api";

type CursorPageState = {
  limit: number;
  currentCursor: string | null;
  backStack: Array<string | null>;
  nextCursor: string | null;
  loading: boolean;
};

type ProjectState = {
  projects: Project[];
  conversationsByProjectId: Record<string, Conversation[]>;
  conversationPagesByProjectId: Record<string, CursorPageState>;
  projectConfigsByProjectId: Record<string, ProjectConfig>;
  projectsPage: CursorPageState;
  activeProjectId: string;
  activeConversationId: string;
  loading: boolean;
  error: string;
};

const initialState: ProjectState = {
  projects: [],
  conversationsByProjectId: {},
  conversationPagesByProjectId: {},
  projectConfigsByProjectId: {},
  projectsPage: createInitialPageState(),
  activeProjectId: "",
  activeConversationId: "",
  loading: false,
  error: ""
};

export const projectStore = reactive<ProjectState>({ ...initialState });

export function resetProjectStore(): void {
  projectStore.projects = [];
  projectStore.conversationsByProjectId = {};
  projectStore.conversationPagesByProjectId = {};
  projectStore.projectConfigsByProjectId = {};
  projectStore.projectsPage = createInitialPageState();
  projectStore.activeProjectId = "";
  projectStore.activeConversationId = "";
  projectStore.loading = false;
  projectStore.error = "";
}

export async function refreshProjects(): Promise<void> {
  await loadProjectsPage({ cursor: null, backStack: [] });
}

export async function loadNextProjectsPage(): Promise<void> {
  const nextCursor = projectStore.projectsPage.nextCursor;
  if (!nextCursor || projectStore.projectsPage.loading) {
    return;
  }

  await loadProjectsPage({
    cursor: nextCursor,
    backStack: [...projectStore.projectsPage.backStack, projectStore.projectsPage.currentCursor]
  });
}

export async function loadPreviousProjectsPage(): Promise<void> {
  if (projectStore.projectsPage.backStack.length === 0 || projectStore.projectsPage.loading) {
    return;
  }

  const backStack = [...projectStore.projectsPage.backStack];
  const previousCursor = backStack.pop() ?? null;
  await loadProjectsPage({ cursor: previousCursor, backStack });
}

async function loadProjectsPage(input: { cursor: string | null; backStack: Array<string | null> }): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  projectStore.loading = true;
  projectStore.projectsPage.loading = true;
  projectStore.error = "";

  try {
    const response = await listProjects(workspace.id, {
      cursor: input.cursor ?? undefined,
      limit: projectStore.projectsPage.limit
    });
    projectStore.projects = response.items;
    const validProjectIDs = new Set(projectStore.projects.map((project) => project.id));
    for (const projectId of Object.keys(projectStore.conversationsByProjectId)) {
      if (!validProjectIDs.has(projectId)) {
        delete projectStore.conversationsByProjectId[projectId];
      }
    }
    for (const projectId of Object.keys(projectStore.conversationPagesByProjectId)) {
      if (!validProjectIDs.has(projectId)) {
        delete projectStore.conversationPagesByProjectId[projectId];
      }
    }
    for (const projectId of Object.keys(projectStore.projectConfigsByProjectId)) {
      if (!validProjectIDs.has(projectId)) {
        delete projectStore.projectConfigsByProjectId[projectId];
      }
    }
    projectStore.projectsPage.currentCursor = input.cursor;
    projectStore.projectsPage.backStack = input.backStack;
    projectStore.projectsPage.nextCursor = response.next_cursor;
    const hasActiveProject = validProjectIDs.has(projectStore.activeProjectId);
    if (!hasActiveProject) {
      projectStore.activeProjectId = projectStore.projects[0]?.id ?? "";
      projectStore.activeConversationId = "";
    } else if (projectStore.activeProjectId === "") {
      projectStore.activeProjectId = projectStore.projects[0]?.id ?? "";
    }
    if (projectStore.activeProjectId === "") {
      projectStore.activeConversationId = "";
      return;
    }
    await refreshWorkspaceProjectConfigs();
    await refreshConversationsForActiveProject();
  } catch (error) {
    projectStore.error = toDisplayError(error);
  } finally {
    projectStore.loading = false;
    projectStore.projectsPage.loading = false;
  }
}

export async function refreshConversationsForActiveProject(): Promise<void> {
  const projectId = projectStore.activeProjectId;
  await loadConversationsPage(projectId, { cursor: null, backStack: [] });
}

export async function loadNextConversationsPage(projectId: string): Promise<void> {
  const page = projectStore.conversationPagesByProjectId[projectId];
  if (!page || !page.nextCursor || page.loading) {
    return;
  }
  await loadConversationsPage(projectId, {
    cursor: page.nextCursor,
    backStack: [...page.backStack, page.currentCursor]
  });
}

export async function loadPreviousConversationsPage(projectId: string): Promise<void> {
  const page = projectStore.conversationPagesByProjectId[projectId];
  if (!page || page.backStack.length === 0 || page.loading) {
    return;
  }
  const backStack = [...page.backStack];
  const previousCursor = backStack.pop() ?? null;
  await loadConversationsPage(projectId, { cursor: previousCursor, backStack });
}

async function loadConversationsPage(projectId: string, input: { cursor: string | null; backStack: Array<string | null> }): Promise<void> {
  if (projectId === "") {
    return;
  }

  const page = ensureConversationPageState(projectId);
  page.loading = true;
  try {
    const response = await listConversations(projectId, {
      cursor: input.cursor ?? undefined,
      limit: page.limit
    });
    projectStore.conversationsByProjectId[projectId] = response.items;
    page.currentCursor = input.cursor;
    page.backStack = input.backStack;
    page.nextCursor = response.next_cursor;
    const hasActiveConversation = response.items.some((conversation) => conversation.id === projectStore.activeConversationId);
    if (!hasActiveConversation) {
      projectStore.activeConversationId = response.items[0]?.id ?? "";
    } else if (projectStore.activeConversationId === "") {
      projectStore.activeConversationId = response.items[0]?.id ?? "";
    }
  } catch (error) {
    projectStore.error = toDisplayError(error);
  } finally {
    page.loading = false;
  }
}

export async function addProject(input: { name: string; repo_path: string; is_git: boolean }): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }

  try {
    const created = await createProject(workspace.id, input);
    projectStore.activeProjectId = created.id;
    projectStore.activeConversationId = "";
    await refreshProjects();
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
    projectStore.activeProjectId = created.id;
    projectStore.activeConversationId = "";
    await refreshProjects();
  } catch (error) {
    projectStore.error = toDisplayError(error);
  }
}

export async function deleteProject(projectId: string): Promise<void> {
  try {
    await removeProject(projectId);
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
  try {
    const created = await createConversation(project, name);
    await refreshConversationsForActiveProject();
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
    await refreshConversationsForActiveProject();
    const list = projectStore.conversationsByProjectId[projectId] ?? [];
    if (projectStore.activeConversationId === conversationId) {
      projectStore.activeConversationId = list[0]?.id ?? "";
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

export async function refreshWorkspaceProjectConfigs(): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }
  try {
    const items = await listWorkspaceProjectConfigs(workspace.id);
    const next: Record<string, ProjectConfig> = {};
    for (const item of items) {
      next[item.project_id] = item.config;
    }
    projectStore.projectConfigsByProjectId = next;
  } catch (error) {
    projectStore.error = toDisplayError(error);
  }
}

export async function refreshProjectConfigById(projectId: string): Promise<void> {
  if (projectId.trim() === "") {
    return;
  }
  try {
    const config = await getProjectConfig(projectId);
    projectStore.projectConfigsByProjectId[projectId] = config;
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

function createInitialPageState(limit = 20): CursorPageState {
  return {
    limit,
    currentCursor: null,
    backStack: [],
    nextCursor: null,
    loading: false
  };
}

function ensureConversationPageState(projectId: string): CursorPageState {
  if (!projectStore.conversationPagesByProjectId[projectId]) {
    projectStore.conversationPagesByProjectId[projectId] = createInitialPageState();
  }
  return projectStore.conversationPagesByProjectId[projectId];
}
