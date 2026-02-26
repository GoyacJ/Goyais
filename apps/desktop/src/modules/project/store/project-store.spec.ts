import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/shared/services/http";
import type { Conversation } from "@/shared/types/api";

const serviceMocks = vi.hoisted(() => ({
  listProjects: vi.fn(),
  createProject: vi.fn(),
  importProjectDirectory: vi.fn(),
  removeProject: vi.fn(),
  listConversations: vi.fn(),
  createConversation: vi.fn(),
  renameConversation: vi.fn(),
  removeConversation: vi.fn(),
  exportConversationMarkdown: vi.fn(),
  updateProjectConfig: vi.fn(),
  getProjectConfig: vi.fn(),
  listWorkspaceProjectConfigs: vi.fn()
}));

const storeMocks = vi.hoisted(() => ({
  getCurrentWorkspace: vi.fn(),
  getWorkspaceToken: vi.fn()
}));

const conversationStoreMocks = vi.hoisted(() => ({
  conversationStore: {
    byConversationId: {} as Record<string, unknown>
  },
  detachConversationStream: vi.fn(),
  clearConversationTimer: vi.fn()
}));

vi.mock("@/modules/project/services", () => ({
  listProjects: serviceMocks.listProjects,
  createProject: serviceMocks.createProject,
  importProjectDirectory: serviceMocks.importProjectDirectory,
  removeProject: serviceMocks.removeProject,
  listConversations: serviceMocks.listConversations,
  createConversation: serviceMocks.createConversation,
  renameConversation: serviceMocks.renameConversation,
  removeConversation: serviceMocks.removeConversation,
  exportConversationMarkdown: serviceMocks.exportConversationMarkdown,
  updateProjectConfig: serviceMocks.updateProjectConfig,
  getProjectConfig: serviceMocks.getProjectConfig,
  listWorkspaceProjectConfigs: serviceMocks.listWorkspaceProjectConfigs
}));

vi.mock("@/shared/stores/workspaceStore", () => ({
  getCurrentWorkspace: storeMocks.getCurrentWorkspace
}));

vi.mock("@/shared/stores/authStore", () => ({
  getWorkspaceToken: storeMocks.getWorkspaceToken
}));

vi.mock("@/modules/conversation/store", () => ({
  conversationStore: conversationStoreMocks.conversationStore,
  detachConversationStream: conversationStoreMocks.detachConversationStream,
  clearConversationTimer: conversationStoreMocks.clearConversationTimer
}));

import { importProjectByDirectory, projectStore, refreshProjects, resetProjectStore, updateProjectBinding } from "@/modules/project/store";

describe("project store token forwarding", () => {
  beforeEach(() => {
    resetProjectStore();
    vi.clearAllMocks();
    conversationStoreMocks.conversationStore.byConversationId = {};

    storeMocks.getCurrentWorkspace.mockReturnValue({
      id: "ws_remote_1",
      mode: "remote"
    });
    storeMocks.getWorkspaceToken.mockReturnValue("at_remote_1");

    serviceMocks.listProjects.mockResolvedValue({
      items: [],
      next_cursor: null
    });
    serviceMocks.listWorkspaceProjectConfigs.mockResolvedValue([]);
    serviceMocks.listConversations.mockResolvedValue({
      items: [],
      next_cursor: null
    });
  });

  it("passes workspace token when refreshing projects in remote workspace", async () => {
    await refreshProjects();

    expect(serviceMocks.listProjects).toHaveBeenCalledWith(
      "ws_remote_1",
      {
        cursor: undefined,
        limit: 20
      },
      { token: "at_remote_1" }
    );
  });

  it("returns created project and passes token when importing project directory", async () => {
    const created = {
      id: "proj_new",
      workspace_id: "ws_remote_1",
      name: "repo-alpha",
      repo_path: "/tmp/repo-alpha",
      is_git: true,
      current_revision: 0,
      created_at: "2026-02-23T00:00:00Z",
      updated_at: "2026-02-23T00:00:00Z"
    };
    serviceMocks.importProjectDirectory.mockResolvedValue(created);
    serviceMocks.listProjects.mockResolvedValue({
      items: [created],
      next_cursor: null
    });

    const result = await importProjectByDirectory("/tmp/repo-alpha");

    expect(serviceMocks.importProjectDirectory).toHaveBeenCalledWith("ws_remote_1", "/tmp/repo-alpha", { token: "at_remote_1" });
    expect(result).toEqual(created);
    expect(projectStore.error).toBe("");
  });

  it("stores readable error message when import fails", async () => {
    serviceMocks.importProjectDirectory.mockRejectedValue(
      new ApiError({
        status: 401,
        code: "ACCESS_DENIED",
        message: "Permission is denied",
        traceId: "tr_import_denied"
      })
    );

    const result = await importProjectByDirectory("/tmp/repo-alpha");

    expect(result).toBeNull();
    expect(projectStore.error).toContain("ACCESS_DENIED");
    expect(projectStore.error).toContain("trace_id: tr_import_denied");
  });

  it("does not auto-select first conversation after project refresh", async () => {
    serviceMocks.listProjects.mockResolvedValue({
      items: [
        {
          id: "proj_alpha",
          workspace_id: "ws_remote_1",
          name: "alpha",
          repo_path: "/tmp/repo-alpha",
          is_git: true,
          default_model_config_id: "rc_model_1",
          default_mode: "agent",
          current_revision: 0,
          created_at: "2026-02-24T00:00:00Z",
          updated_at: "2026-02-24T00:00:00Z"
        }
      ],
      next_cursor: null
    });
    serviceMocks.listConversations.mockResolvedValue({
      items: [
        {
          id: "conv_alpha_1",
          workspace_id: "ws_remote_1",
          project_id: "proj_alpha",
          name: "Conversation 1",
          queue_state: "idle",
          default_mode: "agent",
          model_config_id: "rc_model_1",
          base_revision: 0,
          active_execution_id: null,
          created_at: "2026-02-24T00:00:00Z",
          updated_at: "2026-02-24T00:00:00Z"
        }
      ],
      next_cursor: null
    });

    await refreshProjects();

    expect(projectStore.activeProjectId).toBe("proj_alpha");
    expect(projectStore.activeConversationId).toBe("");
  });

  it("refreshes conversations and clears stale runtime after project binding update", async () => {
    const retainedConversation: Conversation = {
      id: "conv_keep",
      workspace_id: "ws_remote_1",
      project_id: "proj_alpha",
      name: "keep",
      queue_state: "idle",
      default_mode: "agent",
      model_config_id: "rc_model_1",
      rule_ids: [],
      skill_ids: [],
      mcp_ids: [],
      base_revision: 0,
      active_execution_id: null,
      created_at: "2026-02-24T00:00:00Z",
      updated_at: "2026-02-24T00:00:00Z"
    };

    projectStore.conversationsByProjectId.proj_alpha = [
      {
        ...retainedConversation,
        id: "conv_remove",
        name: "remove"
      },
      retainedConversation
    ];
    projectStore.activeConversationId = "conv_remove";
    conversationStoreMocks.conversationStore.byConversationId = {
      conv_remove: { hydrated: true },
      conv_keep: { hydrated: true }
    };

    serviceMocks.updateProjectConfig.mockResolvedValue({
      project_id: "proj_alpha",
      model_config_ids: ["rc_model_1"],
      default_model_config_id: "rc_model_1",
      rule_ids: [],
      skill_ids: [],
      mcp_ids: [],
      updated_at: "2026-02-24T01:00:00Z"
    });
    serviceMocks.listConversations.mockResolvedValue({
      items: [retainedConversation],
      next_cursor: null
    });

    const updated = await updateProjectBinding("proj_alpha", {
      model_config_ids: ["rc_model_1"],
      default_model_config_id: "rc_model_1",
      rule_ids: [],
      skill_ids: [],
      mcp_ids: []
    });

    expect(updated).toBe(true);
    expect(serviceMocks.updateProjectConfig).toHaveBeenCalledTimes(1);
    expect(serviceMocks.listConversations).toHaveBeenCalledTimes(1);
    expect(conversationStoreMocks.detachConversationStream).toHaveBeenCalledWith("conv_remove");
    expect(conversationStoreMocks.clearConversationTimer).toHaveBeenCalledWith("conv_remove");
    expect(conversationStoreMocks.conversationStore.byConversationId).not.toHaveProperty("conv_remove");
    expect(projectStore.activeConversationId).toBe("");
    expect(projectStore.conversationsByProjectId.proj_alpha).toHaveLength(1);
    expect(projectStore.conversationsByProjectId.proj_alpha[0]?.id).toBe("conv_keep");
  });

  it("keeps modal state clean when project binding update fails", async () => {
    projectStore.projectConfigsByProjectId.proj_alpha = {
      project_id: "proj_alpha",
      model_config_ids: ["rc_model_1"],
      default_model_config_id: "rc_model_1",
      rule_ids: [],
      skill_ids: [],
      mcp_ids: [],
      updated_at: "2026-02-24T00:00:00Z"
    };

    serviceMocks.updateProjectConfig.mockRejectedValue(
      new ApiError({
        status: 400,
        code: "VALIDATION_ERROR",
        message: "model_config_id must be included in project model_config_ids",
        traceId: "tr_bind_invalid"
      })
    );

    const updated = await updateProjectBinding("proj_alpha", {
      model_config_ids: ["rc_model_1"],
      default_model_config_id: "rc_model_1",
      rule_ids: [],
      skill_ids: [],
      mcp_ids: []
    });

    expect(updated).toBe(false);
    expect(projectStore.projectConfigsByProjectId.proj_alpha?.updated_at).toBe("2026-02-24T00:00:00Z");
    expect(projectStore.error).toContain("VALIDATION_ERROR");
    expect(projectStore.error).toContain("tr_bind_invalid");
    expect(serviceMocks.listConversations).not.toHaveBeenCalled();
  });
});
