import { getControlClient } from "@/shared/services/clients";
import { withApiFallback } from "@/shared/services/fallback";
import { createMockId, mockData } from "@/shared/services/mockData";
import type { Conversation, ListEnvelope, Project, ProjectConfig } from "@/shared/types/api";

export async function listProjects(workspaceId: string): Promise<ListEnvelope<Project>> {
  return withApiFallback(
    "project.list",
    () => getControlClient().get<ListEnvelope<Project>>(`/v1/projects?workspace_id=${encodeURIComponent(workspaceId)}`),
    () => ({
      items: mockData.projects.filter((project) => project.workspace_id === workspaceId),
      next_cursor: null
    })
  );
}

export async function createProject(workspaceId: string, input: { name: string; repo_path: string; is_git: boolean }): Promise<Project> {
  return withApiFallback(
    "project.create",
    () =>
      getControlClient().post<Project>("/v1/projects", {
        workspace_id: workspaceId,
        ...input
      }),
    () => {
      const now = new Date().toISOString();
      const created: Project = {
        id: createMockId("proj"),
        workspace_id: workspaceId,
        name: input.name,
        repo_path: input.repo_path,
        is_git: input.is_git,
        default_mode: "agent",
        default_model_id: "gpt-4.1",
        created_at: now,
        updated_at: now
      };
      mockData.projects.push(created);
      return created;
    }
  );
}

export async function importProjectDirectory(workspaceId: string, repoPath: string): Promise<Project> {
  return withApiFallback(
    "project.import",
    () =>
      getControlClient().post<Project>("/v1/projects/import", {
        workspace_id: workspaceId,
        directory_path: repoPath
      }),
    () => {
      const now = new Date().toISOString();
      const folderName = repoPath.split("/").filter(Boolean).pop() ?? "Imported Project";
      const created: Project = {
        id: createMockId("proj"),
        workspace_id: workspaceId,
        name: folderName,
        repo_path: repoPath,
        is_git: true,
        default_mode: "agent",
        default_model_id: "gpt-4.1",
        created_at: now,
        updated_at: now
      };
      mockData.projects.push(created);
      return created;
    }
  );
}

export async function removeProject(projectId: string): Promise<void> {
  return withApiFallback(
    "project.remove",
    async () => {
      await getControlClient().request<void>(`/v1/projects/${projectId}`, { method: "DELETE" });
    },
    () => {
      mockData.projects = mockData.projects.filter((project) => project.id !== projectId);
      mockData.conversations = mockData.conversations.filter((conversation) => conversation.project_id !== projectId);
    }
  );
}

export async function listConversations(projectId: string): Promise<ListEnvelope<Conversation>> {
  return withApiFallback(
    "project.listConversations",
    () => getControlClient().get<ListEnvelope<Conversation>>(`/v1/projects/${projectId}/conversations`),
    () => ({
      items: mockData.conversations.filter((conversation) => conversation.project_id === projectId),
      next_cursor: null
    })
  );
}

export async function createConversation(project: Project, name: string): Promise<Conversation> {
  return withApiFallback(
    "project.createConversation",
    () =>
      getControlClient().post<Conversation>(`/v1/projects/${project.id}/conversations`, {
        workspace_id: project.workspace_id,
        name
      }),
    () => {
      const now = new Date().toISOString();
      const created: Conversation = {
        id: createMockId("conv"),
        workspace_id: project.workspace_id,
        project_id: project.id,
        name,
        queue_state: "idle",
        default_mode: project.default_mode ?? "agent",
        model_id: project.default_model_id ?? "gpt-4.1",
        active_execution_id: null,
        created_at: now,
        updated_at: now
      };
      mockData.conversations.push(created);
      return created;
    }
  );
}

export async function renameConversation(conversationId: string, name: string): Promise<Conversation> {
  return withApiFallback(
    "project.renameConversation",
    () => getControlClient().request<Conversation>(`/v1/conversations/${conversationId}`, { method: "PATCH", body: { name } }),
    () => {
      const target = mockData.conversations.find((conversation) => conversation.id === conversationId);
      if (!target) {
        throw new Error("Conversation not found");
      }
      target.name = name;
      target.updated_at = new Date().toISOString();
      return target;
    }
  );
}

export async function removeConversation(conversationId: string): Promise<void> {
  return withApiFallback(
    "project.removeConversation",
    async () => {
      await getControlClient().request<void>(`/v1/conversations/${conversationId}`, { method: "DELETE" });
    },
    () => {
      mockData.conversations = mockData.conversations.filter((conversation) => conversation.id !== conversationId);
    }
  );
}

export async function exportConversationMarkdown(conversationId: string): Promise<string> {
  return withApiFallback(
    "project.exportConversationMarkdown",
    () => getControlClient().get<string>(`/v1/conversations/${conversationId}/export?format=markdown`),
    () => {
      return `# Conversation ${conversationId}\n\n- Export format: markdown\n- Generated at: ${new Date().toISOString()}\n`;
    }
  );
}

export async function updateProjectConfig(projectId: string, config: Omit<ProjectConfig, "project_id" | "updated_at">): Promise<ProjectConfig> {
  return withApiFallback(
    "project.updateConfig",
    () => getControlClient().request<ProjectConfig>(`/v1/projects/${projectId}/config`, { method: "PUT", body: config }),
    () => ({
      project_id: projectId,
      model_id: config.model_id,
      rule_ids: [...config.rule_ids],
      skill_ids: [...config.skill_ids],
      mcp_ids: [...config.mcp_ids],
      updated_at: new Date().toISOString()
    })
  );
}
