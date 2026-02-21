import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/lib/api-error";
import type { WorkspaceProfile } from "@/stores/workspaceStore";

const runtimeClientMock = vi.hoisted(() => ({
  createRun: vi.fn(),
  confirmToolCall: vi.fn(),
  listRuns: vi.fn(),
  listSessions: vi.fn(),
  createSession: vi.fn(),
  renameSession: vi.fn(),
  replayRunEvents: vi.fn(),
  runtimeHealth: vi.fn(),
  subscribeRunEvents: vi.fn()
}));

const runtimeGatewayClientMock = vi.hoisted(() => ({
  createRun: vi.fn(),
  confirmToolCall: vi.fn(),
  listRuns: vi.fn(),
  listSessions: vi.fn(),
  createSession: vi.fn(),
  renameSession: vi.fn(),
  replayRunEvents: vi.fn(),
  runtimeHealth: vi.fn(),
  subscribeRunEvents: vi.fn()
}));

const secretStoreMock = vi.hoisted(() => ({
  loadToken: vi.fn()
}));

vi.mock("@/api/runtimeClient", () => runtimeClientMock);
vi.mock("@/api/runtimeGatewayClient", () => runtimeGatewayClientMock);
vi.mock("@/api/secretStoreClient", () => secretStoreMock);

import { getRunDataSource } from "@/api/runDataSource";

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
    id: "remote-profile-1",
    kind: "remote",
    name: "Remote A",
    remote: {
      serverUrl: "http://127.0.0.1:8787",
      tokenRef: "remote-profile-1",
      selectedWorkspaceId: "ws-1"
    },
    lastUsedAt: new Date().toISOString()
  };
}

describe("runDataSource", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("uses local runtime client when workspace is local", async () => {
    runtimeClientMock.runtimeHealth.mockResolvedValue({ ok: true });
    const source = getRunDataSource(makeLocalProfile());

    await source.runtimeHealth();
    expect(source.kind).toBe("local");
    expect(runtimeClientMock.runtimeHealth).toHaveBeenCalledTimes(1);
    expect(runtimeGatewayClientMock.runtimeHealth).not.toHaveBeenCalled();
  });

  it("uses hub runtime gateway when workspace is remote", async () => {
    secretStoreMock.loadToken.mockResolvedValue("token-abc");
    runtimeGatewayClientMock.createRun.mockResolvedValue({ run_id: "run-1" });

    const source = getRunDataSource(makeRemoteProfile());
    await source.createRun({
      project_id: "p1",
      session_id: "s1",
      input: "task",
      model_config_id: "mc1",
      workspace_path: "/tmp/work",
      options: { use_worktree: false }
    });

    expect(runtimeGatewayClientMock.createRun).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-abc",
      "ws-1",
      expect.objectContaining({
        project_id: "p1"
      })
    );
  });

  it("fails with unauthorized when remote token is missing", async () => {
    secretStoreMock.loadToken.mockResolvedValue(null);
    const source = getRunDataSource(makeRemoteProfile());

    const request = source.runtimeHealth();
    await expect(request).rejects.toBeInstanceOf(ApiError);
    await expect(request).rejects.toMatchObject({ code: "E_UNAUTHORIZED" });
  });

  it("routes remote confirmations through gateway strict endpoint", async () => {
    secretStoreMock.loadToken.mockResolvedValue("token-xyz");
    runtimeGatewayClientMock.confirmToolCall.mockResolvedValue(undefined);

    const source = getRunDataSource(makeRemoteProfile());
    await source.confirmToolCall("run-7", "call-9", true);

    expect(runtimeGatewayClientMock.confirmToolCall).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-xyz",
      "ws-1",
      "run-7",
      "call-9",
      true
    );
  });

  it("routes session list/create/rename through the active data source", async () => {
    secretStoreMock.loadToken.mockResolvedValue("token-sessions");
    runtimeGatewayClientMock.listSessions.mockResolvedValue({ sessions: [] });
    runtimeGatewayClientMock.createSession.mockResolvedValue({
      session: {
        session_id: "s1",
        project_id: "p1",
        title: "Thread",
        updated_at: "2026-02-20T00:00:00.000Z"
      }
    });
    runtimeGatewayClientMock.renameSession.mockResolvedValue({
      session: {
        session_id: "s1",
        project_id: "p1",
        title: "Renamed",
        updated_at: "2026-02-20T00:00:00.000Z"
      }
    });

    const source = getRunDataSource(makeRemoteProfile());
    await source.listSessions("p1");
    await source.createSession({ project_id: "p1", title: "Thread" });
    await source.renameSession("s1", "Renamed");

    expect(runtimeGatewayClientMock.listSessions).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-sessions",
      "ws-1",
      "p1"
    );
    expect(runtimeGatewayClientMock.createSession).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-sessions",
      "ws-1",
      { project_id: "p1", title: "Thread" }
    );
    expect(runtimeGatewayClientMock.renameSession).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-sessions",
      "ws-1",
      "s1",
      "Renamed"
    );
  });
});
