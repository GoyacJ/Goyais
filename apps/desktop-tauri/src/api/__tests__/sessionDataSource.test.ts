import { beforeEach, describe, expect, it, vi } from "vitest";

import type { WorkspaceProfile } from "@/stores/workspaceStore";

const hubClientMock = vi.hoisted(() => ({
  getHealth: vi.fn(),
  listWorkspaces: vi.fn(),
  listSessions: vi.fn(),
  createSession: vi.fn(),
  updateSession: vi.fn(),
  archiveSession: vi.fn(),
  executeSession: vi.fn(),
  cancelExecution: vi.fn(),
  decideConfirmation: vi.fn(),
  subscribeSessionEvents: vi.fn(),
  getRuntimeHealth: vi.fn(),
  commitExecution: vi.fn(),
  exportExecutionPatch: vi.fn(),
  discardExecution: vi.fn(),
  bootstrapAdmin: vi.fn(),
  getBootstrapStatus: vi.fn(),
  login: vi.fn()
}));

const secretStoreMock = vi.hoisted(() => ({
  loadToken: vi.fn(),
  deleteToken: vi.fn()
}));

const tauriCoreMock = vi.hoisted(() => ({
  invoke: vi.fn()
}));

vi.mock("@/api/hubClient", () => hubClientMock);
vi.mock("@/api/secretStoreClient", () => secretStoreMock);
vi.mock("@tauri-apps/api/core", () => tauriCoreMock);

import { ensureLocalHubContext, getSessionDataSource } from "@/api/sessionDataSource";
import { ApiError } from "@/lib/api-error";

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
    delete (window as Window & { __TAURI_INTERNALS__?: unknown }).__TAURI_INTERNALS__;
    secretStoreMock.deleteToken.mockResolvedValue(undefined);
  });

  it("resolves local hub context without auth/login flow", async () => {
    localStorage.setItem("goyais.localAutoPassword", "legacy-password");
    hubClientMock.getHealth.mockResolvedValue({ status: "ok", version: "0.2.0" });
    hubClientMock.listWorkspaces.mockResolvedValue({
      workspaces: [{ workspace_id: "ws-local", name: "Local", slug: "local", role_name: "owner" }]
    });

    const ctx = await ensureLocalHubContext();

    expect(ctx).toEqual({
      serverUrl: "http://127.0.0.1:8787",
      token: "",
      workspaceId: "ws-local"
    });
    expect(hubClientMock.listWorkspaces).toHaveBeenCalledWith("http://127.0.0.1:8787", "");
    expect(hubClientMock.getBootstrapStatus).not.toHaveBeenCalled();
    expect(hubClientMock.bootstrapAdmin).not.toHaveBeenCalled();
    expect(hubClientMock.login).not.toHaveBeenCalled();
    expect(localStorage.getItem("goyais.localAutoPassword")).toBeNull();
    expect(secretStoreMock.deleteToken).toHaveBeenCalledWith("local-default");
  });

  it("uses runtime health gateway in local mode", async () => {
    hubClientMock.getHealth.mockResolvedValue({ status: "ok", version: "0.2.0" });
    hubClientMock.listWorkspaces.mockResolvedValue({
      workspaces: [{ workspace_id: "ws-local", name: "Local", slug: "local", role_name: "owner" }]
    });
    hubClientMock.getRuntimeHealth.mockResolvedValue({
      workspace_id: "ws-local",
      runtime_status: "online",
      upstream: { ok: true }
    });

    const source = getSessionDataSource(makeLocalProfile());
    const payload = await source.runtimeHealth();

    expect(source.kind).toBe("local");
    expect(payload.ok).toBe(true);
    expect(hubClientMock.getRuntimeHealth).toHaveBeenCalledWith("http://127.0.0.1:8787", "", "ws-local");
  });

  it("returns unauthorized when remote token is missing", async () => {
    secretStoreMock.loadToken.mockResolvedValue(null);

    const source = getSessionDataSource(makeRemoteProfile());
    const request = source.listSessions("p-1");

    await expect(request).rejects.toBeInstanceOf(ApiError);
    await expect(request).rejects.toMatchObject({ code: "E_UNAUTHORIZED" });
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

  it("throws network error when local hub is unreachable", async () => {
    hubClientMock.getHealth.mockRejectedValue(new Error("Network request failed"));

    const request = ensureLocalHubContext();
    await expect(request).rejects.toBeInstanceOf(ApiError);
    await expect(request).rejects.toMatchObject({ code: "NETWORK_OR_RUNTIME_ERROR" });
  });
});
