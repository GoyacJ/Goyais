import { getControlClient } from "@/shared/services/clients";
import { withApiFallback } from "@/shared/services/fallback";
import { createMockId, mockData } from "@/shared/services/mockData";
import type {
  Conversation,
  ConversationMode,
  ListEnvelope,
  PaginationQuery,
  Project,
  ProjectConfig,
  WorkspaceProjectConfigItem
} from "@/shared/types/api";

type ProjectServiceOptions = {
  token?: string;
};

export async function listProjects(
  workspaceId: string,
  query: PaginationQuery = {},
  options: ProjectServiceOptions = {}
): Promise<ListEnvelope<Project>> {
  const search = buildPaginationSearch({ ...query, workspace_id: workspaceId });
  return getControlClient().get<ListEnvelope<Project>>(`/v1/projects${search}`, { token: options.token });
}

export async function createProject(
  workspaceId: string,
  input: { name: string; repo_path: string; is_git: boolean },
  options: ProjectServiceOptions = {}
): Promise<Project> {
  return withApiFallback(
    "project.create",
    () =>
      getControlClient().post<Project>("/v1/projects", {
        workspace_id: workspaceId,
        ...input
      }, { token: options.token }),
    () => {
      const now = new Date().toISOString();
      const created: Project = {
        id: createMockId("proj"),
        workspace_id: workspaceId,
        name: input.name,
        repo_path: input.repo_path,
        is_git: input.is_git,
        default_mode: "agent",
        default_model_id: resolveMockDefaultModelID(),
        current_revision: 0,
        created_at: now,
        updated_at: now
      };
      mockData.projects.push(created);
      return created;
    }
  );
}

export async function importProjectDirectory(
  workspaceId: string,
  repoPath: string,
  options: ProjectServiceOptions = {}
): Promise<Project> {
  return getControlClient().post<Project>("/v1/projects/import", {
    workspace_id: workspaceId,
    directory_path: repoPath
  }, { token: options.token });
}

export async function removeProject(projectId: string, options: ProjectServiceOptions = {}): Promise<void> {
  await getControlClient().request<void>(`/v1/projects/${projectId}`, { method: "DELETE", token: options.token });
}

export async function listConversations(
  projectId: string,
  query: PaginationQuery = {},
  options: ProjectServiceOptions = {}
): Promise<ListEnvelope<Conversation>> {
  const search = buildPaginationSearch(query);
  return withApiFallback(
    "project.listConversations",
    () => getControlClient().get<ListEnvelope<Conversation>>(`/v1/projects/${projectId}/conversations${search}`, { token: options.token }),
    () => paginateMock(mockData.conversations.filter((conversation) => conversation.project_id === projectId), query)
  );
}

export async function createConversation(project: Project, name: string, options: ProjectServiceOptions = {}): Promise<Conversation> {
  return withApiFallback(
    "project.createConversation",
    () =>
      getControlClient().post<Conversation>(`/v1/projects/${project.id}/conversations`, {
        workspace_id: project.workspace_id,
        name
      }, { token: options.token }),
    () => {
      const now = new Date().toISOString();
      const created: Conversation = {
        id: createMockId("conv"),
        workspace_id: project.workspace_id,
        project_id: project.id,
        name,
        queue_state: "idle",
        default_mode: project.default_mode ?? "agent",
        model_id: resolveMockDefaultModelID(project),
        base_revision: project.current_revision ?? 0,
        active_execution_id: null,
        created_at: now,
        updated_at: now
      };
      mockData.conversations.push(created);
      return created;
    }
  );
}

export async function renameConversation(
  conversationId: string,
  name: string,
  options: ProjectServiceOptions = {}
): Promise<Conversation> {
  return patchConversation(conversationId, { name }, options);
}

export async function patchConversation(
  conversationId: string,
  patch: { name?: string; mode?: ConversationMode; model_id?: string },
  options: ProjectServiceOptions = {}
): Promise<Conversation> {
  return withApiFallback(
    "project.patchConversation",
    () => getControlClient().request<Conversation>(`/v1/conversations/${conversationId}`, { method: "PATCH", body: patch, token: options.token }),
    () => {
      const target = mockData.conversations.find((conversation) => conversation.id === conversationId);
      if (!target) {
        throw new Error("Conversation not found");
      }
      if (patch.name !== undefined) {
        target.name = patch.name;
      }
      if (patch.mode !== undefined) {
        target.default_mode = patch.mode;
      }
      if (patch.model_id !== undefined) {
        target.model_id = patch.model_id;
      }
      target.updated_at = new Date().toISOString();
      return target;
    }
  );
}

export async function removeConversation(conversationId: string, options: ProjectServiceOptions = {}): Promise<void> {
  return withApiFallback(
    "project.removeConversation",
    async () => {
      await getControlClient().request<void>(`/v1/conversations/${conversationId}`, { method: "DELETE", token: options.token });
    },
    () => {
      mockData.conversations = mockData.conversations.filter((conversation) => conversation.id !== conversationId);
    }
  );
}

export async function exportConversationMarkdown(conversationId: string, options: ProjectServiceOptions = {}): Promise<string> {
  return withApiFallback(
    "project.exportConversationMarkdown",
    () => getControlClient().get<string>(`/v1/conversations/${conversationId}/export?format=markdown`, { token: options.token }),
    () => {
      return `# Conversation ${conversationId}\n\n- Export format: markdown\n- Generated at: ${new Date().toISOString()}\n`;
    }
  );
}

export async function updateProjectConfig(
  projectId: string,
  config: Omit<ProjectConfig, "project_id" | "updated_at">,
  options: ProjectServiceOptions = {}
): Promise<ProjectConfig> {
  return getControlClient().request<ProjectConfig>(`/v1/projects/${projectId}/config`, { method: "PUT", body: config, token: options.token });
}

export async function getProjectConfig(projectId: string, options: ProjectServiceOptions = {}): Promise<ProjectConfig> {
  return getControlClient().get<ProjectConfig>(`/v1/projects/${projectId}/config`, { token: options.token });
}

export async function listWorkspaceProjectConfigs(
  workspaceId: string,
  options: ProjectServiceOptions = {}
): Promise<WorkspaceProjectConfigItem[]> {
  return getControlClient().get<WorkspaceProjectConfigItem[]>(`/v1/workspaces/${workspaceId}/project-configs`, { token: options.token });
}

function buildPaginationSearch(query: PaginationQuery & { workspace_id?: string }): string {
  const params = new URLSearchParams();
  if (query.workspace_id) {
    params.set("workspace_id", query.workspace_id);
  }
  if (query.cursor) {
    params.set("cursor", query.cursor);
  }
  if (query.limit !== undefined) {
    params.set("limit", String(query.limit));
  }
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}

function paginateMock<T>(items: T[], query: PaginationQuery): ListEnvelope<T> {
  const start = Number.parseInt(query.cursor ?? "0", 10);
  const safeStart = Number.isNaN(start) || start < 0 ? 0 : start;
  const limit = query.limit !== undefined && query.limit > 0 ? query.limit : 20;
  const end = Math.min(safeStart + limit, items.length);
  return {
    items: items.slice(safeStart, end),
    next_cursor: end < items.length ? String(end) : null
  };
}

function resolveMockDefaultModelID(project?: Project): string {
  const projectDefaultModelID = project?.default_model_id?.trim();
  if (projectDefaultModelID) {
    return projectDefaultModelID;
  }
  const resourceConfig = mockData.resourceConfigs.find(
    (item) => item.type === "model" && item.enabled && typeof item.model?.model_id === "string" && item.model.model_id.trim() !== ""
  );
  return resourceConfig?.model?.model_id?.trim() ?? "";
}
