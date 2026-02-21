import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/lib/api-error";
import type { WorkspaceProfile } from "@/stores/workspaceStore";

const hubClientMock = vi.hoisted(() => ({
  listProjects: vi.fn(),
  createProject: vi.fn(),
  deleteProject: vi.fn(),
  syncProject: vi.fn(),
  listModelConfigs: vi.fn(),
  createModelConfig: vi.fn(),
  updateModelConfig: vi.fn(),
  deleteModelConfig: vi.fn(),
  listRuntimeModelCatalog: vi.fn()
}));

const sessionDataSourceMock = vi.hoisted(() => ({
  resolveHubContext: vi.fn()
}));

vi.mock("@/api/hubClient", () => hubClientMock);
vi.mock("@/api/sessionDataSource", () => sessionDataSourceMock);

import { getModelConfigsClient, getProjectsClient } from "@/api/dataSource";

function makeLocalProfile(): WorkspaceProfile {
  return {
    id: "local-default",
    kind: "local",
    name: "Local Workspace",
    local: {
      rootPath: "/Users/goya/Repo/Git/Goyais"
    },
    lastUsedAt: new Date().toISOString()
  };
}

function makeRemoteProfile(): WorkspaceProfile {
  return {
    id: "profile-1",
    kind: "remote",
    name: "Remote A",
    remote: {
      serverUrl: "http://127.0.0.1:8787",
      tokenRef: "profile-1",
      selectedWorkspaceId: "ws-1"
    },
    lastUsedAt: new Date().toISOString()
  };
}

describe("dataSource", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it("uses hub client for local projects", async () => {
    sessionDataSourceMock.resolveHubContext.mockResolvedValue({
      serverUrl: "http://127.0.0.1:8787",
      token: "",
      workspaceId: "ws-local"
    });
    hubClientMock.listProjects.mockResolvedValue({
      projects: [{ project_id: "p-local", workspace_id: "ws-local", name: "Local Project", root_uri: "/tmp/local" }]
    });

    const client = getProjectsClient(makeLocalProfile());
    const result = await client.list();

    expect(client.kind).toBe("local");
    expect(hubClientMock.listProjects).toHaveBeenCalledWith("http://127.0.0.1:8787", "", "ws-local");
    expect(result[0]?.workspace_path).toBe("/tmp/local");
  });

  it("supports deleting local projects via hub", async () => {
    sessionDataSourceMock.resolveHubContext.mockResolvedValue({
      serverUrl: "http://127.0.0.1:8787",
      token: "",
      workspaceId: "ws-local"
    });

    const client = getProjectsClient(makeLocalProfile());
    await client.delete("p-local");

    expect(client.supportsDelete).toBe(true);
    expect(hubClientMock.deleteProject).toHaveBeenCalledWith("http://127.0.0.1:8787", "", "ws-local", "p-local");
  });

  it("uses hub client for remote projects", async () => {
    sessionDataSourceMock.resolveHubContext.mockResolvedValue({
      serverUrl: "http://127.0.0.1:8787",
      token: "token-abc",
      workspaceId: "ws-1"
    });
    hubClientMock.listProjects.mockResolvedValue({
      projects: [
        {
          project_id: "p-1",
          workspace_id: "ws-1",
          name: "Remote Project",
          root_uri: "repo://demo/main",
          created_at: "2026-02-20T00:00:00.000Z",
          updated_at: "2026-02-20T00:00:00.000Z"
        }
      ]
    });

    const client = getProjectsClient(makeRemoteProfile());
    const result = await client.list();

    expect(client.kind).toBe("remote");
    expect(hubClientMock.listProjects).toHaveBeenCalledWith("http://127.0.0.1:8787", "token-abc", "ws-1");
    expect(result[0]?.root_uri).toBe("repo://demo/main");
  });

  it("does not write api_key into localStorage on create", async () => {
    localStorage.setItem("workspace", "stable");
    sessionDataSourceMock.resolveHubContext.mockResolvedValue({
      serverUrl: "http://127.0.0.1:8787",
      token: "token-abc",
      workspaceId: "ws-1"
    });
    hubClientMock.createModelConfig.mockResolvedValue({
      model_config: {
        model_config_id: "mc-1",
        workspace_id: "ws-1",
        provider: "openai",
        model: "gpt-4.1-mini",
        base_url: null,
        temperature: 0,
        max_tokens: 1024,
        secret_ref: "secret:mc-1",
        created_at: "2026-02-20T00:00:00.000Z",
        updated_at: "2026-02-20T00:00:00.000Z"
      }
    });

    const client = getModelConfigsClient(makeRemoteProfile());
    await client.create({
      provider: "openai",
      model: "gpt-4.1-mini",
      api_key: "sk-live-123"
    });

    expect(localStorage.getItem("workspace")).toBe("stable");
    expect(localStorage.getItem("sk-live-123")).toBeNull();
    expect(hubClientMock.createModelConfig).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-abc",
      "ws-1",
      expect.objectContaining({
        api_key: "sk-live-123"
      })
    );
  });

  it("requires api_key when creating model config", async () => {
    sessionDataSourceMock.resolveHubContext.mockResolvedValue({
      serverUrl: "http://127.0.0.1:8787",
      token: "",
      workspaceId: "ws-local"
    });

    const client = getModelConfigsClient(makeLocalProfile());
    const request = client.create({
      provider: "openai",
      model: "gpt-4.1-mini"
    });

    await expect(request).rejects.toBeInstanceOf(ApiError);
    await expect(request).rejects.toMatchObject({ code: "E_VALIDATION" });
  });

  it("requests model catalog from hub runtime gateway and forwards api override", async () => {
    sessionDataSourceMock.resolveHubContext.mockResolvedValue({
      serverUrl: "http://127.0.0.1:8787",
      token: "token-abc",
      workspaceId: "ws-1"
    });
    hubClientMock.listRuntimeModelCatalog.mockResolvedValue({
      provider: "openai",
      items: [],
      fetched_at: "2026-02-20T00:00:00.000Z",
      fallback_used: false
    });

    const client = getModelConfigsClient(makeRemoteProfile());
    await client.listModels("mc-1", { apiKeyOverride: "sk-override" });

    expect(hubClientMock.listRuntimeModelCatalog).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-abc",
      "ws-1",
      "mc-1",
      { apiKeyOverride: "sk-override" }
    );
  });
});
