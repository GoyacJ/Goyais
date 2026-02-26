import {
  getProjectConfig,
  listConversations,
  listProjects,
  listWorkspaceProjectConfigs
} from "@/modules/project/services";
import { toDisplayError } from "@/shared/services/errorMapper";
import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";

import {
  ensureConversationPageState,
  projectStore,
  resolveCurrentWorkspaceToken,
  resolveWorkspaceToken
} from "@/modules/project/store/state";
import type { ProjectConfig } from "@/shared/types/api";

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
  const token = resolveWorkspaceToken(workspace.id);

  projectStore.loading = true;
  projectStore.projectsPage.loading = true;
  projectStore.error = "";

  try {
    const response = await listProjects(
      workspace.id,
      {
        cursor: input.cursor ?? undefined,
        limit: projectStore.projectsPage.limit
      },
      { token }
    );
    projectStore.projects = response.items;
    const validProjectIDs = new Set(projectStore.projects.map((project) => project.id));
    pruneRemovedProjects(validProjectIDs);
    projectStore.projectsPage.currentCursor = input.cursor;
    projectStore.projectsPage.backStack = input.backStack;
    projectStore.projectsPage.nextCursor = response.next_cursor;
    syncActiveProject(validProjectIDs);
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

function pruneRemovedProjects(validProjectIDs: Set<string>): void {
  removeMissingKeys(projectStore.conversationsByProjectId, validProjectIDs);
  removeMissingKeys(projectStore.conversationPagesByProjectId, validProjectIDs);
  removeMissingKeys(projectStore.projectConfigsByProjectId, validProjectIDs);
}

function removeMissingKeys(record: Record<string, unknown>, validProjectIDs: Set<string>): void {
  for (const key of Object.keys(record)) {
    if (!validProjectIDs.has(key)) {
      delete record[key];
    }
  }
}

function syncActiveProject(validProjectIDs: Set<string>): void {
  const fallbackProjectID = projectStore.projects[0]?.id ?? "";
  if (!validProjectIDs.has(projectStore.activeProjectId)) {
    projectStore.activeProjectId = fallbackProjectID;
    projectStore.activeConversationId = "";
    return;
  }
  if (projectStore.activeProjectId === "") {
    projectStore.activeProjectId = fallbackProjectID;
  }
}

export async function refreshConversationsForActiveProject(): Promise<void> {
  await refreshConversationsForProject(projectStore.activeProjectId);
}

export async function refreshConversationsForProject(projectId: string): Promise<void> {
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

  const token = resolveCurrentWorkspaceToken();
  const page = ensureConversationPageState(projectId);
  page.loading = true;
  try {
    const response = await listConversations(
      projectId,
      {
        cursor: input.cursor ?? undefined,
        limit: page.limit
      },
      { token }
    );
    projectStore.conversationsByProjectId[projectId] = response.items;
    page.currentCursor = input.cursor;
    page.backStack = input.backStack;
    page.nextCursor = response.next_cursor;
    const hasActiveConversation = response.items.some((conversation) => conversation.id === projectStore.activeConversationId);
    if (!hasActiveConversation) {
      projectStore.activeConversationId = "";
    }
  } catch (error) {
    projectStore.error = toDisplayError(error);
  } finally {
    page.loading = false;
  }
}

export async function refreshWorkspaceProjectConfigs(): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }
  const token = resolveWorkspaceToken(workspace.id);
  try {
    const items = await listWorkspaceProjectConfigs(workspace.id, { token });
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
  const token = resolveCurrentWorkspaceToken();
  try {
    const config = await getProjectConfig(projectId, { token });
    projectStore.projectConfigsByProjectId[projectId] = config;
  } catch (error) {
    projectStore.error = toDisplayError(error);
  }
}
