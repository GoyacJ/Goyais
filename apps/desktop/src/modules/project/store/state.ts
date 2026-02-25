import { defineStore } from "pinia";

import { getWorkspaceToken } from "@/shared/stores/authStore";
import { pinia } from "@/shared/stores/pinia";
import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";
import type { Conversation, Project, ProjectConfig } from "@/shared/types/api";

export type CursorPageState = {
  limit: number;
  currentCursor: string | null;
  backStack: Array<string | null>;
  nextCursor: string | null;
  loading: boolean;
};

export type ProjectState = {
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

const useProjectStoreDefinition = defineStore("project", {
  state: (): ProjectState => ({ ...initialState })
});

export const useProjectStore = useProjectStoreDefinition;
export const projectStore = useProjectStoreDefinition(pinia);

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

export function createInitialPageState(limit = 20): CursorPageState {
  return {
    limit,
    currentCursor: null,
    backStack: [],
    nextCursor: null,
    loading: false
  };
}

export function ensureConversationPageState(projectId: string): CursorPageState {
  if (!projectStore.conversationPagesByProjectId[projectId]) {
    projectStore.conversationPagesByProjectId[projectId] = createInitialPageState();
  }
  return projectStore.conversationPagesByProjectId[projectId];
}

export function resolveCurrentWorkspaceToken(): string | undefined {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return undefined;
  }
  return resolveWorkspaceToken(workspace.id);
}

export function resolveWorkspaceToken(workspaceId: string): string | undefined {
  const token = getWorkspaceToken(workspaceId).trim();
  if (token === "") {
    return undefined;
  }
  return token;
}
