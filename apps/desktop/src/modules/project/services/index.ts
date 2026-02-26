import { getControlClient } from "@/shared/services/clients";
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
  return getControlClient().post<Project>(
    "/v1/projects",
    {
      workspace_id: workspaceId,
      ...input
    },
    { token: options.token }
  );
}

export async function importProjectDirectory(
  workspaceId: string,
  repoPath: string,
  options: ProjectServiceOptions = {}
): Promise<Project> {
  return getControlClient().post<Project>(
    "/v1/projects/import",
    {
      workspace_id: workspaceId,
      directory_path: repoPath
    },
    { token: options.token }
  );
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
  return getControlClient().get<ListEnvelope<Conversation>>(`/v1/projects/${projectId}/conversations${search}`, {
    token: options.token
  });
}

export async function createConversation(project: Project, name: string, options: ProjectServiceOptions = {}): Promise<Conversation> {
  return getControlClient().post<Conversation>(
    `/v1/projects/${project.id}/conversations`,
    {
      workspace_id: project.workspace_id,
      name
    },
    { token: options.token }
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
  patch: {
    name?: string;
    mode?: ConversationMode;
    model_config_id?: string;
    rule_ids?: string[];
    skill_ids?: string[];
    mcp_ids?: string[];
  },
  options: ProjectServiceOptions = {}
): Promise<Conversation> {
  return getControlClient().request<Conversation>(`/v1/conversations/${conversationId}`, {
    method: "PATCH",
    body: patch,
    token: options.token
  });
}

export async function removeConversation(conversationId: string, options: ProjectServiceOptions = {}): Promise<void> {
  await getControlClient().request<void>(`/v1/conversations/${conversationId}`, {
    method: "DELETE",
    token: options.token
  });
}

export async function exportConversationMarkdown(conversationId: string, options: ProjectServiceOptions = {}): Promise<string> {
  return getControlClient().get<string>(`/v1/conversations/${conversationId}/export?format=markdown`, {
    token: options.token
  });
}

export async function updateProjectConfig(
  projectId: string,
  config: Omit<ProjectConfig, "project_id" | "updated_at">,
  options: ProjectServiceOptions = {}
): Promise<ProjectConfig> {
  return getControlClient().request<ProjectConfig>(`/v1/projects/${projectId}/config`, {
    method: "PUT",
    body: config,
    token: options.token
  });
}

export async function getProjectConfig(projectId: string, options: ProjectServiceOptions = {}): Promise<ProjectConfig> {
  return getControlClient().get<ProjectConfig>(`/v1/projects/${projectId}/config`, { token: options.token });
}

export async function listWorkspaceProjectConfigs(
  workspaceId: string,
  options: ProjectServiceOptions = {}
): Promise<WorkspaceProjectConfigItem[]> {
  return getControlClient().get<WorkspaceProjectConfigItem[]>(`/v1/workspaces/${workspaceId}/project-configs`, {
    token: options.token
  });
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
