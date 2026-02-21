import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/lib/api-error";
import type { WorkspaceProfile } from "@/stores/workspaceStore";

const hubClientMock = vi.hoisted(() => ({
  listWorkspaces: vi.fn(),
  listSessions: vi.fn(),
  createSession: vi.fn(),
  updateSession: vi.fn(),
  archiveSession: vi.fn(),
  executeSession: vi.fn(),
  cancelExecution: vi.fn(),
  decideConfirmation: vi.fn(),
  subscribeSessionEvents: vi.fn(),
  getHealth: vi.fn(),
  commitExecution: vi.fn(),
  exportExecutionPatch: vi.fn(),
  discardExecution: vi.fn()
}));

const secretStoreMock = vi.hoisted(() => ({
  loadToken: vi.fn()
}));

vi.mock("@/api/hubClient", () => hubClientMock);
vi.mock("@/api/secretStoreClient", () => secretStoreMock);

import { getSessionDataSource } from "@/api/sessionDataSource";

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

describe("sessionDataSource", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it("uses hub health endpoint in local mode", async () => {
    secretStoreMock.loadToken.mockResolvedValue("token-local");
    hubClientMock.listWorkspaces.mockResolvedValue({
      workspaces: [{ workspace_id: "ws-local", name: "Local", slug: "local", role_name: "owner" }]
    });
    hubClientMock.getHealth.mockResolvedValue({ status: "ok", version: "0.2.0" });

    const source = getSessionDataSource(makeLocalProfile());
    const payload = await source.runtimeHealth();

    expect(source.kind).toBe("local");
    expect(payload.ok).toBe(true);
    expect(hubClientMock.getHealth).toHaveBeenCalledTimes(1);
    expect(String(hubClientMock.getHealth.mock.calls[0][0])).toContain("http://127.0.0.1");
  });

  it("uses hub execution endpoint when workspace is remote", async () => {
    secretStoreMock.loadToken.mockResolvedValue("token-abc");
    hubClientMock.executeSession.mockResolvedValue({
      execution_id: "exec-1",
      trace_id: "trace-1",
      session_id: "s1",
      state: "executing"
    });

    const source = getSessionDataSource(makeRemoteProfile());
    const response = await source.executeSession("s1", "task");

    expect(response.execution_id).toBe("exec-1");
    expect(hubClientMock.executeSession).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-abc",
      "ws-1",
      "s1",
      "task"
    );
  });

  it("fails with unauthorized when remote token is missing", async () => {
    secretStoreMock.loadToken.mockResolvedValue(null);
    const source = getSessionDataSource(makeRemoteProfile());

    const request = source.runtimeHealth();
    await expect(request).rejects.toBeInstanceOf(ApiError);
    await expect(request).rejects.toMatchObject({ code: "E_UNAUTHORIZED" });
  });

  it("routes cancel and confirmation decisions through hub", async () => {
    secretStoreMock.loadToken.mockResolvedValue("token-xyz");
    hubClientMock.cancelExecution.mockResolvedValue(undefined);
    hubClientMock.decideConfirmation.mockResolvedValue(undefined);

    const source = getSessionDataSource(makeRemoteProfile());
    await source.cancelExecution("exec-7");
    await source.decideConfirmation("exec-7", "call-9", "approved");

    expect(hubClientMock.cancelExecution).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-xyz",
      "ws-1",
      "exec-7"
    );
    expect(hubClientMock.decideConfirmation).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-xyz",
      "ws-1",
      "exec-7",
      "call-9",
      "approved"
    );
  });

  it("routes session list/create/rename through hub", async () => {
    secretStoreMock.loadToken.mockResolvedValue("token-sessions");
    hubClientMock.listSessions.mockResolvedValue({ sessions: [] });
    hubClientMock.createSession.mockResolvedValue({
      session: {
        session_id: "s1",
        project_id: "p1",
        workspace_id: "ws-1",
        title: "Thread",
        mode: "agent",
        status: "idle",
        updated_at: "2026-02-20T00:00:00.000Z"
      }
    });
    hubClientMock.updateSession.mockResolvedValue({
      session: {
        session_id: "s1",
        project_id: "p1",
        workspace_id: "ws-1",
        title: "Renamed",
        mode: "agent",
        status: "idle",
        updated_at: "2026-02-20T00:00:00.000Z"
      }
    });

    const source = getSessionDataSource(makeRemoteProfile());
    await source.listSessions("p1");
    await source.createSession({ project_id: "p1", title: "Thread" });
    await source.renameSession("s1", "Renamed");

    expect(hubClientMock.listSessions).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-sessions",
      "ws-1",
      "p1"
    );
    expect(hubClientMock.createSession).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-sessions",
      "ws-1",
      { project_id: "p1", title: "Thread", mode: "agent", model_config_id: null, use_worktree: true }
    );
    expect(hubClientMock.updateSession).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-sessions",
      "ws-1",
      "s1",
      { title: "Renamed" }
    );
  });

  it("routes execution commit/export/discard through hub", async () => {
    secretStoreMock.loadToken.mockResolvedValue("token-exec");
    hubClientMock.commitExecution.mockResolvedValue({ commit_sha: "abc123" });
    hubClientMock.exportExecutionPatch.mockResolvedValue("--- a/README.md\n+++ b/README.md\n");
    hubClientMock.discardExecution.mockResolvedValue(undefined);

    const source = getSessionDataSource(makeRemoteProfile());
    const commitResult = await source.commitExecution("exec-1", "feat: update");
    const patch = await source.exportExecutionPatch("exec-1");
    await source.discardExecution("exec-1");

    expect(commitResult.commit_sha).toBe("abc123");
    expect(patch).toContain("--- a/README.md");
    expect(hubClientMock.commitExecution).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-exec",
      "ws-1",
      "exec-1",
      "feat: update"
    );
    expect(hubClientMock.exportExecutionPatch).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-exec",
      "ws-1",
      "exec-1"
    );
    expect(hubClientMock.discardExecution).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-exec",
      "ws-1",
      "exec-1"
    );
  });

  it("subscribes to session events with resolved hub context", async () => {
    secretStoreMock.loadToken.mockResolvedValue("token-sub");
    hubClientMock.subscribeSessionEvents.mockReturnValue(() => undefined);

    const source = getSessionDataSource(makeRemoteProfile());
    const sub = source.subscribeSessionEvents("s1", 12, vi.fn(), vi.fn());

    await Promise.resolve();
    await Promise.resolve();

    expect(hubClientMock.subscribeSessionEvents).toHaveBeenCalledWith(
      "http://127.0.0.1:8787",
      "token-sub",
      "ws-1",
      "s1",
      12,
      expect.any(Function),
      expect.any(Function)
    );

    sub.close();
  });
});
