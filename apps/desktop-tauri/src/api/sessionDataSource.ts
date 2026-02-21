import * as hubClient from "@/api/hubClient";
import {
  deleteToken,
  loadLocalHubCredentials,
  loadToken,
  type LocalHubCredentials,
  storeLocalHubCredentials,
  storeToken
} from "@/api/secretStoreClient";
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

export const LOCAL_HUB_STORAGE_KEY = "goyais.localHubUrl";
export const DEFAULT_LOCAL_HUB_URL = import.meta.env.VITE_LOCAL_HUB_URL ?? "http://127.0.0.1:8080";
export const LOCAL_TOKEN_PROFILE_ID = "local-default";
export const LOCAL_WORKSPACE_STORAGE_KEY = "goyais.localWorkspaceId";
const LOCAL_AUTO_EMAIL = "local-admin@goyais.local";
const LOCAL_AUTO_DISPLAY_NAME = "Local Admin";
const LOCAL_UNLOCK_REQUIRED_CODE = "E_LOCAL_HUB_UNLOCK_REQUIRED";
const LOCAL_UNLOCK_REQUIRED_MESSAGE = "Local workspace needs one-time unlock. Please enter local account credentials.";

export function localHubBaseUrl(): string {
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

export interface EnsureLocalHubAuthOptions {
  unlockCredentials?: {
    email: string;
    password: string;
    displayName?: string;
  };
}

function normalizeLocalCredentials(credentials: {
  email: string;
  password: string;
  displayName?: string;
}): LocalHubCredentials {
  return {
    email: credentials.email.trim().toLowerCase(),
    password: credentials.password,
    displayName: credentials.displayName?.trim() || LOCAL_AUTO_DISPLAY_NAME
  };
}

function generateLocalPassword(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return `${crypto.randomUUID()}${crypto.randomUUID()}`;
  }
  return `local-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function createAutoLocalCredentials(): LocalHubCredentials {
  return {
    email: LOCAL_AUTO_EMAIL,
    password: generateLocalPassword(),
    displayName: LOCAL_AUTO_DISPLAY_NAME
  };
}

function isUnauthorized(error: unknown): boolean {
  return error instanceof ApiError && error.status === 401;
}

function isSetupCompletedConflict(error: unknown): boolean {
  return error instanceof ApiError && (
    error.status === 409
    || error.code === "E_ALREADY_BOOTSTRAPPED"
    || error.code === "E_SETUP_COMPLETED"
  );
}

function buildLocalUnlockRequiredError(): ApiError {
  return toError({
    code: LOCAL_UNLOCK_REQUIRED_CODE,
    message: LOCAL_UNLOCK_REQUIRED_MESSAGE,
    status: 401
  });
}

function pickWorkspaceId(workspaces: Array<{ workspace_id: string }>): string {
  const cachedWorkspaceId = localStorage.getItem(LOCAL_WORKSPACE_STORAGE_KEY);
  const selectedWorkspaceId =
    cachedWorkspaceId && workspaces.some((workspace) => workspace.workspace_id === cachedWorkspaceId)
      ? cachedWorkspaceId
      : workspaces[0]?.workspace_id;
  if (!selectedWorkspaceId) {
    throw toError({ code: "E_NO_WORKSPACE", message: "No workspace found on local hub." });
  }
  localStorage.setItem(LOCAL_WORKSPACE_STORAGE_KEY, selectedWorkspaceId);
  return selectedWorkspaceId;
}

async function resolveWorkspaceId(serverUrl: string, token: string): Promise<string> {
  const resp = await hubClient.listWorkspaces(serverUrl, token);
  return pickWorkspaceId(resp.workspaces);
}

export async function ensureLocalHubAuth(options: EnsureLocalHubAuthOptions = {}): Promise<HubContext> {
  const serverUrl = localHubBaseUrl();
  const providedCredentials = options.unlockCredentials
    ? normalizeLocalCredentials(options.unlockCredentials)
    : null;

  const existingToken = await loadToken(LOCAL_TOKEN_PROFILE_ID);
  if (existingToken) {
    try {
      const workspaceId = await resolveWorkspaceId(serverUrl, existingToken);
      return { serverUrl, token: existingToken, workspaceId };
    } catch (error) {
      if (!isUnauthorized(error)) {
        throw error;
      }
      await deleteToken(LOCAL_TOKEN_PROFILE_ID).catch(() => undefined);
      localStorage.removeItem(LOCAL_WORKSPACE_STORAGE_KEY);
    }
  }

  const status = await hubClient.getBootstrapStatus(serverUrl);

  if (status.setup_mode) {
    const savedCredentials = await loadLocalHubCredentials();
    const bootstrapCredentials = providedCredentials ?? savedCredentials ?? createAutoLocalCredentials();

    try {
      const bootstrapResponse = await hubClient.bootstrapAdmin(serverUrl, {
        email: bootstrapCredentials.email,
        password: bootstrapCredentials.password,
        display_name: bootstrapCredentials.displayName
      });
      await storeToken(LOCAL_TOKEN_PROFILE_ID, bootstrapResponse.token);
      await storeLocalHubCredentials(bootstrapCredentials);
      const workspaceId = bootstrapResponse.workspace?.workspace_id ?? await resolveWorkspaceId(serverUrl, bootstrapResponse.token);
      localStorage.setItem(LOCAL_WORKSPACE_STORAGE_KEY, workspaceId);
      return { serverUrl, token: bootstrapResponse.token, workspaceId };
    } catch (error) {
      if (!isSetupCompletedConflict(error)) {
        throw error;
      }
    }
  }

  const loginCredentials = providedCredentials ?? await loadLocalHubCredentials();
  if (!loginCredentials) {
    throw buildLocalUnlockRequiredError();
  }

  try {
    const loginResponse = await hubClient.login(serverUrl, {
      email: loginCredentials.email,
      password: loginCredentials.password
    });
    await storeToken(LOCAL_TOKEN_PROFILE_ID, loginResponse.token);
    await storeLocalHubCredentials(loginCredentials);
    const workspaceId = await resolveWorkspaceId(serverUrl, loginResponse.token);
    return { serverUrl, token: loginResponse.token, workspaceId };
  } catch (error) {
    if (providedCredentials) {
      throw error;
    }
    if (isUnauthorized(error)) {
      throw buildLocalUnlockRequiredError();
    }
    throw error;
  }
}

async function resolveLocalHubContext(): Promise<HubContext> {
  return ensureLocalHubAuth();
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
