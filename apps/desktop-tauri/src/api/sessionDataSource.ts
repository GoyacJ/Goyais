import * as hubClient from "@/api/hubClient";
import { loadToken } from "@/api/secretStoreClient";
import { ApiError } from "@/lib/api-error";
import type { WorkspaceProfile } from "@/stores/workspaceStore";

export interface SessionSummary {
  session_id: string;
  project_id: string;
  workspace_id: string;
  title: string;
  mode: "plan" | "agent";
  status: "idle" | "executing" | "waiting_confirmation";
  updated_at: string;
}

export interface SessionEventsSubscription {
  close: () => void;
}

export interface SessionDataSource {
  kind: "local" | "remote";
  listSessions: (projectId: string) => Promise<{ sessions: SessionSummary[] }>;
  createSession: (payload: {
    project_id: string;
    title?: string;
    mode?: "plan" | "agent";
    model_config_id?: string | null;
    use_worktree?: boolean;
  }) => Promise<{ session: SessionSummary }>;
  renameSession: (sessionId: string, title: string) => Promise<{ session: SessionSummary }>;
  archiveSession: (sessionId: string) => Promise<void>;
  executeSession: (sessionId: string, message: string) => Promise<hubClient.ExecutionResponse>;
  cancelExecution: (executionId: string) => Promise<void>;
  decideConfirmation: (executionId: string, callId: string, decision: "approved" | "denied") => Promise<void>;
  subscribeSessionEvents: (
    sessionId: string,
    sinceSeq: number,
    onEvent: (type: string, payloadJson: string, seq: number) => void,
    onError?: (error: Error) => void
  ) => SessionEventsSubscription;
  runtimeHealth: () => Promise<{ ok: boolean }>;
  commitExecution: (executionId: string, message?: string) => Promise<{ commit_sha: string }>;
  exportExecutionPatch: (executionId: string) => Promise<string>;
  discardExecution: (executionId: string) => Promise<void>;
}

const LOCAL_HUB_STORAGE_KEY = "goyais.localHubUrl";
const DEFAULT_LOCAL_HUB_URL = import.meta.env.VITE_LOCAL_HUB_URL ?? "http://127.0.0.1:8080";
const LOCAL_TOKEN_PROFILE_ID = "local-default";
const LOCAL_WORKSPACE_STORAGE_KEY = "goyais.localWorkspaceId";

function localHubBaseUrl(): string {
  return localStorage.getItem(LOCAL_HUB_STORAGE_KEY) ?? DEFAULT_LOCAL_HUB_URL;
}

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

export interface HubContext {
  serverUrl: string;
  token: string;
  workspaceId: string;
}

async function resolveLocalHubContext(): Promise<HubContext> {
  const serverUrl = localHubBaseUrl();
  const token = await loadToken(LOCAL_TOKEN_PROFILE_ID);
  if (!token) {
    throw toError({
      code: "E_LOCAL_HUB_NOT_BOOTSTRAPPED",
      message: "Local hub not yet bootstrapped. Please restart the app.",
      status: 401
    });
  }

  const cachedWorkspaceId = localStorage.getItem(LOCAL_WORKSPACE_STORAGE_KEY);
  if (cachedWorkspaceId) {
    return { serverUrl, token, workspaceId: cachedWorkspaceId };
  }

  const resp = await hubClient.listWorkspaces(serverUrl, token);
  const workspace = resp.workspaces[0];
  if (!workspace) {
    throw toError({ code: "E_NO_WORKSPACE", message: "No workspace found on local hub." });
  }

  localStorage.setItem(LOCAL_WORKSPACE_STORAGE_KEY, workspace.workspace_id);
  return { serverUrl, token, workspaceId: workspace.workspace_id };
}

async function resolveRemoteHubContext(profile: WorkspaceProfile): Promise<HubContext> {
  if (profile.kind !== "remote" || !profile.remote) {
    throw toError({ code: "E_VALIDATION", message: "Remote workspace profile required", status: 400 });
  }

  const workspaceId = profile.remote.selectedWorkspaceId;
  if (!workspaceId) {
    throw toError({ code: "E_VALIDATION", message: "Remote workspace not selected", status: 400 });
  }

  const tokenRef = profile.remote.tokenRef || profile.id;
  const token = await loadToken(tokenRef);
  if (!token) {
    throw toError({ code: "E_UNAUTHORIZED", message: "Token not found. Please login again.", status: 401 });
  }

  return { serverUrl: profile.remote.serverUrl, token, workspaceId };
}

function mapHubSession(s: hubClient.HubSession): SessionSummary {
  return {
    session_id: s.session_id,
    project_id: s.project_id,
    workspace_id: s.workspace_id,
    title: s.title,
    mode: s.mode,
    status: s.status,
    updated_at: s.updated_at
  };
}

function makeHubSessionDataSource(
  kind: "local" | "remote",
  resolveCtx: () => Promise<HubContext>
): SessionDataSource {
  return {
    kind,

    listSessions: async (projectId) => {
      const ctx = await resolveCtx();
      const resp = await hubClient.listSessions(ctx.serverUrl, ctx.token, ctx.workspaceId, projectId);
      return { sessions: resp.sessions.map(mapHubSession) };
    },

    createSession: async ({ project_id, title, mode, model_config_id, use_worktree }) => {
      const ctx = await resolveCtx();
      const resp = await hubClient.createSession(ctx.serverUrl, ctx.token, ctx.workspaceId, {
        project_id,
        title: title ?? "New Session",
        mode: mode ?? "agent",
        model_config_id: model_config_id ?? null,
        use_worktree: use_worktree ?? true
      });
      return { session: mapHubSession(resp.session) };
    },

    renameSession: async (sessionId, title) => {
      const ctx = await resolveCtx();
      const resp = await hubClient.updateSession(ctx.serverUrl, ctx.token, ctx.workspaceId, sessionId, { title });
      return { session: mapHubSession(resp.session) };
    },

    archiveSession: async (sessionId) => {
      const ctx = await resolveCtx();
      await hubClient.archiveSession(ctx.serverUrl, ctx.token, ctx.workspaceId, sessionId);
    },

    executeSession: async (sessionId, message) => {
      const ctx = await resolveCtx();
      return hubClient.executeSession(ctx.serverUrl, ctx.token, ctx.workspaceId, sessionId, message);
    },

    cancelExecution: async (executionId) => {
      const ctx = await resolveCtx();
      await hubClient.cancelExecution(ctx.serverUrl, ctx.token, ctx.workspaceId, executionId);
    },

    decideConfirmation: async (executionId, callId, decision) => {
      const ctx = await resolveCtx();
      await hubClient.decideConfirmation(ctx.serverUrl, ctx.token, ctx.workspaceId, executionId, callId, decision);
    },

    subscribeSessionEvents: (sessionId, sinceSeq, onEvent, onError) => {
      let closed = false;
      let cleanup: (() => void) | undefined;

      void (async () => {
        try {
          const ctx = await resolveCtx();
          if (closed) {
            return;
          }

          cleanup = hubClient.subscribeSessionEvents(
            ctx.serverUrl,
            ctx.token,
            ctx.workspaceId,
            sessionId,
            sinceSeq,
            onEvent,
            onError
          );
        } catch (error) {
          onError?.(error as Error);
        }
      })();

      return {
        close: () => {
          closed = true;
          cleanup?.();
        }
      };
    },

    runtimeHealth: async () => {
      const ctx = await resolveCtx();
      const payload = await hubClient.getHealth(ctx.serverUrl);
      return { ok: payload.status === "ok" };
    },

    commitExecution: async (executionId, message) => {
      const ctx = await resolveCtx();
      return hubClient.commitExecution(ctx.serverUrl, ctx.token, ctx.workspaceId, executionId, message);
    },

    exportExecutionPatch: async (executionId) => {
      const ctx = await resolveCtx();
      return hubClient.exportExecutionPatch(ctx.serverUrl, ctx.token, ctx.workspaceId, executionId);
    },

    discardExecution: async (executionId) => {
      const ctx = await resolveCtx();
      await hubClient.discardExecution(ctx.serverUrl, ctx.token, ctx.workspaceId, executionId);
    }
  };
}

export async function resolveHubContext(profile: WorkspaceProfile | undefined): Promise<HubContext> {
  if (!profile || profile.kind === "local") {
    return resolveLocalHubContext();
  }
  return resolveRemoteHubContext(profile);
}

export function getSessionDataSource(profile: WorkspaceProfile | undefined): SessionDataSource {
  if (!profile || profile.kind === "local") {
    return makeHubSessionDataSource("local", resolveLocalHubContext);
  }

  return makeHubSessionDataSource("remote", () => resolveRemoteHubContext(profile));
}
