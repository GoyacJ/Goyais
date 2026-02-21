import * as runtimeClient from "@/api/runtimeClient";
import * as runtimeGatewayClient from "@/api/runtimeGatewayClient";
import { loadToken } from "@/api/secretStoreClient";
import { ApiError } from "@/lib/api-error";
import type { WorkspaceProfile } from "@/stores/workspaceStore";
import type { EventEnvelope } from "@/types/generated";

function toError(shape: {
  code: string;
  message: string;
  retryable?: boolean;
  status?: number;
}): ApiError {
  return new ApiError({
    code: shape.code,
    message: shape.message,
    retryable: shape.retryable ?? false,
    status: shape.status
  });
}

async function resolveRemoteContext(profile: WorkspaceProfile): Promise<{
  workspaceId: string;
  serverUrl: string;
  token: string;
}> {
  if (profile.kind !== "remote" || !profile.remote) {
    throw toError({
      code: "E_VALIDATION",
      message: "Remote workspace profile is required",
      status: 400
    });
  }

  const workspaceId = profile.remote.selectedWorkspaceId;
  if (!workspaceId) {
    throw toError({
      code: "E_VALIDATION",
      message: "Remote workspace is not selected",
      status: 400
    });
  }

  const tokenRef = profile.remote.tokenRef || profile.id;
  const token = await loadToken(tokenRef);
  if (!token) {
    throw toError({
      code: "E_UNAUTHORIZED",
      message: "Token not found in keychain. Please login again.",
      status: 401
    });
  }

  return {
    workspaceId,
    serverUrl: profile.remote.serverUrl,
    token
  };
}

export interface RunEventsSubscription {
  close: () => void;
}

export interface RunDataSource {
  kind: "local" | "remote";
  createRun: (payload: runtimeClient.RunCreateRequest) => Promise<{ run_id: string }>;
  confirmToolCall: (runId: string, callId: string, approved: boolean) => Promise<void>;
  listRuns: (sessionId: string) => Promise<{ runs: Array<Record<string, string>> }>;
  listSessions: (projectId: string) => Promise<{ sessions: runtimeClient.RuntimeSessionSummary[] }>;
  createSession: (payload: { project_id: string; title?: string }) => Promise<{ session: runtimeClient.RuntimeSessionSummary }>;
  renameSession: (sessionId: string, title: string) => Promise<{ session: runtimeClient.RuntimeSessionSummary }>;
  replayRunEvents: (runId: string) => Promise<{ events: EventEnvelope[] }>;
  runtimeHealth: () => Promise<{ ok: boolean }>;
  subscribeRunEvents: (
    runId: string,
    onEvent: (event: EventEnvelope) => void,
    onError?: (error: Error) => void
  ) => RunEventsSubscription;
}

export function getRunDataSource(profile: WorkspaceProfile | undefined): RunDataSource {
  if (!profile || profile.kind === "local") {
    return {
      kind: "local",
      createRun: (payload) => runtimeClient.createRun(payload),
      confirmToolCall: (runId, callId, approved) => runtimeClient.confirmToolCall(runId, callId, approved),
      listRuns: (sessionId) => runtimeClient.listRuns(sessionId),
      listSessions: (projectId) => runtimeClient.listSessions(projectId),
      createSession: (payload) => runtimeClient.createSession(payload),
      renameSession: (sessionId, title) => runtimeClient.renameSession(sessionId, title),
      replayRunEvents: (runId) => runtimeClient.replayRunEvents(runId),
      runtimeHealth: () => runtimeClient.runtimeHealth(),
      subscribeRunEvents: (runId, onEvent) => {
        const source = runtimeClient.subscribeRunEvents(runId, onEvent);
        return {
          close() {
            source.close();
          }
        };
      }
    };
  }

  return {
    kind: "remote",
    createRun: async (payload) => {
      const remote = await resolveRemoteContext(profile);
      return runtimeGatewayClient.createRun(remote.serverUrl, remote.token, remote.workspaceId, payload);
    },
    confirmToolCall: async (runId, callId, approved) => {
      const remote = await resolveRemoteContext(profile);
      return runtimeGatewayClient.confirmToolCall(remote.serverUrl, remote.token, remote.workspaceId, runId, callId, approved);
    },
    listRuns: async (sessionId) => {
      const remote = await resolveRemoteContext(profile);
      return runtimeGatewayClient.listRuns(remote.serverUrl, remote.token, remote.workspaceId, sessionId);
    },
    listSessions: async (projectId) => {
      const remote = await resolveRemoteContext(profile);
      return runtimeGatewayClient.listSessions(remote.serverUrl, remote.token, remote.workspaceId, projectId);
    },
    createSession: async (payload) => {
      const remote = await resolveRemoteContext(profile);
      return runtimeGatewayClient.createSession(remote.serverUrl, remote.token, remote.workspaceId, payload);
    },
    renameSession: async (sessionId, title) => {
      const remote = await resolveRemoteContext(profile);
      return runtimeGatewayClient.renameSession(remote.serverUrl, remote.token, remote.workspaceId, sessionId, title);
    },
    replayRunEvents: async (runId) => {
      const remote = await resolveRemoteContext(profile);
      return runtimeGatewayClient.replayRunEvents(remote.serverUrl, remote.token, remote.workspaceId, runId);
    },
    runtimeHealth: async () => {
      const remote = await resolveRemoteContext(profile);
      const payload = await runtimeGatewayClient.runtimeHealth(remote.serverUrl, remote.token, remote.workspaceId);
      return { ok: payload.runtime_status === "online" && payload.upstream?.ok === true };
    },
    subscribeRunEvents: (runId, onEvent, onError) => {
      let closed = false;
      let delegate: RunEventsSubscription | undefined;
      void (async () => {
        try {
          const remote = await resolveRemoteContext(profile);
          if (closed) {
            return;
          }
          delegate = runtimeGatewayClient.subscribeRunEvents(
            remote.serverUrl,
            remote.token,
            remote.workspaceId,
            runId,
            onEvent,
            onError
          );
        } catch (error) {
          onError?.(error as Error);
        }
      })();

      return {
        close() {
          closed = true;
          delegate?.close();
        }
      };
    }
  };
}
