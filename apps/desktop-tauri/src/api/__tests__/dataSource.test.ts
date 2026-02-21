import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/lib/api-error";
import type { WorkspaceProfile } from "@/stores/workspaceStore";

const runtimeClientMock = vi.hoisted(() => ({
  listProjects: vi.fn(),
  createProject: vi.fn(),
  listModelConfigs: vi.fn(),
  createModelConfig: vi.fn(),
  updateModelConfig: vi.fn(),
  deleteModelConfig: vi.fn(),
  listModelCatalog: vi.fn()
}));

const hubClientMock = vi.hoisted(() => ({
  listProjects: vi.fn(),
  createProject: vi.fn(),
  deleteProject: vi.fn(),
  listModelConfigs: vi.fn(),
  createModelConfig: vi.fn(),
  updateModelConfig: vi.fn(),
  deleteModelConfig: vi.fn(),
  listRuntimeModelCatalog: vi.fn()
}));

const secretStoreMock = vi.hoisted(() => ({
  loadToken: vi.fn()
}));

vi.mock("@/api/runtimeClient", () => runtimeClientMock);
vi.mock("@/api/hubClient", () => hubClientMock);
vi.mock("@/api/secretStoreClient", () => secretStoreMock);

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

  it("uses runtime client for local projects", async () => {
    runtimeClientMock.listProjects.mockResolvedValue({
      projects: [{ project_id: "p-local", name: "Local Project", workspace_path: "/tmp/local" }]
    });

    const client = getProjectsClient(makeLocalProfile());
    const result = await client.list();

    expect(client.kind).toBe("local");
    expect(runtimeClientMock.listProjects).toHaveBeenCalledTimes(1);
    expect(hubClientMock.listProjects).not.toHaveBeenCalled();
    expect(result[0]?.workspace_path).toBe("/tmp/local");
  });

  it("uses hub client for remote projects", async () => {
    secretStoreMock.loadToken.mockResolvedValue("token-abc");
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
    expect(secretStoreMock.loadToken).toHaveBeenCalledWith("profile-1");
    expect(hubClientMock.listProjects).toHaveBeenCalledWith("http://127.0.0.1:8787", "token-abc", "ws-1");
    expect(result[0]?.root_uri).toBe("repo://demo/main");
  });

  it("throws unauthorized when remote token is missing", async () => {
    secretStoreMock.loadToken.mockResolvedValue(null);

    const client = getProjectsClient(makeRemoteProfile());
    const request = client.list();
    await expect(request).rejects.toBeInstanceOf(ApiError);
    await expect(request).rejects.toMatchObject({ code: "E_UNAUTHORIZED" });
  });

  it("does not write api_key into localStorage on remote create", async () => {
    localStorage.setItem("workspace", "stable");
    secretStoreMock.loadToken.mockResolvedValue("token-abc");
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

  it("requests model catalog from runtime gateway for remote profile", async () => {
    secretStoreMock.loadToken.mockResolvedValue("token-abc");
    hubClientMock.listRuntimeModelCatalog.mockResolvedValue({
      provider: "openai",
      items: [],
      fetched_at: "2026-02-20T00:00:00.000Z",
      fallback_used: false
    });

    const client = getModelConfigsClient(makeRemoteProfile());
    await client.listModels("mc-1");

    expect(hubClientMock.listRuntimeModelCatalog).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-abc",
      "ws-1",
      "mc-1"
    );
  });
});
