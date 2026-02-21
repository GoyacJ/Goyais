import * as hubClient from "@/api/hubClient";
import { invoke } from "@tauri-apps/api/core";
import { deleteToken, loadToken } from "@/api/secretStoreClient";
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
export const DEFAULT_LOCAL_HUB_URL = import.meta.env.VITE_LOCAL_HUB_URL ?? "http://127.0.0.1:8787";
export const LOCAL_TOKEN_PROFILE_ID = "local-default";
export const LOCAL_WORKSPACE_STORAGE_KEY = "goyais.localWorkspaceId";
const LEGACY_LOCAL_AUTO_PASSWORD_STORAGE_KEY = "goyais.localAutoPassword";
const WORKSPACE_STORE_KEY = "goyais.workspace.registry.v1";
const LOCAL_HUB_START_WAIT_MS = 6_000;
const LOCAL_HUB_START_POLL_MS = 250;

let localHubStartInFlight: Promise<boolean> | null = null;
let localLegacyCleanupDone = false;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function normalizeHubUrl(url: string): string {
  return url.trim().replace(/\/+$/, "");
}

function withHubUrlScheme(url: string): string {
  const normalized = normalizeHubUrl(url);
  if (normalized.startsWith("http://") || normalized.startsWith("https://")) {
    return normalized;
  }
  return `http://${normalized}`;
}

function dedupeUrls(urls: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const url of urls) {
    if (seen.has(url)) {
      continue;
    }
    seen.add(url);
    out.push(url);
  }
  return out;
}

function buildLocalHubCandidates(preferredUrl: string): string[] {
  return dedupeUrls([preferredUrl, withHubUrlScheme(DEFAULT_LOCAL_HUB_URL)]);
}

function persistLocalHubBaseUrl(serverUrl: string) {
  localStorage.setItem(LOCAL_HUB_STORAGE_KEY, withHubUrlScheme(serverUrl));
}

function inferWorkspaceRootFromStore(): string | null {
  const raw = localStorage.getItem(WORKSPACE_STORE_KEY);
  if (!raw) {
    return null;
  }

  try {
    const parsed = JSON.parse(raw);
    const snapshots: Array<{ profiles?: Array<{ id?: string; kind?: string; local?: { rootPath?: string } }>; currentProfileId?: string }> = [];

    if (isRecord(parsed)) {
      snapshots.push(parsed);
      if (isRecord(parsed.state)) {
        snapshots.push(parsed.state);
      }
    }

    for (const snapshot of snapshots) {
      const profiles = Array.isArray(snapshot.profiles) ? snapshot.profiles : [];
      if (profiles.length === 0) {
        continue;
      }

      const byCurrentId = snapshot.currentProfileId
        ? profiles.find((profile) => profile.id === snapshot.currentProfileId && profile.kind === "local")
        : undefined;
      const localProfile = byCurrentId ?? profiles.find((profile) => profile.kind === "local");
      const rootPath = localProfile?.local?.rootPath?.trim();
      if (rootPath) {
        return rootPath;
      }
    }

    return null;
  } catch {
    return null;
  }
}

function buildLocalHubStartCommand(): string {
  const custom = import.meta.env.VITE_LOCAL_HUB_START_COMMAND as string | undefined;
  if (custom && custom.trim()) {
    return custom.trim();
  }

  return [
    "if command -v goyais-hub >/dev/null 2>&1; then",
    "  PORT=8787 GOYAIS_AUTH_MODE=local_open goyais-hub",
    "elif [ -f ./server/hub-server-go/cmd/hub/main.go ] && command -v go >/dev/null 2>&1; then",
    "  cd ./server/hub-server-go && PORT=8787 GOYAIS_AUTH_MODE=local_open GOYAIS_DB_PATH=./data/hub.db go run cmd/hub/main.go",
    "elif [ -f ./package.json ] && command -v pnpm >/dev/null 2>&1; then",
    "  PORT=8787 GOYAIS_AUTH_MODE=local_open pnpm dev:hub",
    "else",
    "  exit 127",
    "fi"
  ].join("\n");
}

function resolveLocalHubStartCwd(): string {
  const configured = import.meta.env.VITE_LOCAL_HUB_START_CWD as string | undefined;
  if (configured && configured.trim()) {
    return configured.trim();
  }
  return inferWorkspaceRootFromStore() ?? ".";
}

async function delay(ms: number): Promise<void> {
  await new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

async function isLocalHubReachable(serverUrl: string): Promise<boolean> {
  try {
    await hubClient.getHealth(serverUrl);
    return true;
  } catch {
    return false;
  }
}

async function waitForLocalHub(candidates: string[]): Promise<boolean> {
  const deadline = Date.now() + LOCAL_HUB_START_WAIT_MS;
  while (Date.now() < deadline) {
    for (const candidate of candidates) {
      if (await isLocalHubReachable(candidate)) {
        return true;
      }
    }
    await delay(LOCAL_HUB_START_POLL_MS);
  }
  return false;
}

function isTauriRuntime(): boolean {
  if (typeof window === "undefined") {
    return false;
  }
  return Object.prototype.hasOwnProperty.call(window, "__TAURI_INTERNALS__");
}

async function ensureLocalHubProcessRunning(candidates: string[]): Promise<boolean> {
  if (!isTauriRuntime()) {
    return false;
  }

  if (!localHubStartInFlight) {
    localHubStartInFlight = (async () => {
      try {
        const existingPid = await invoke<number | null>("runtime_status");
        if (!existingPid) {
          await invoke<number>("runtime_start", {
            command: buildLocalHubStartCommand(),
            cwd: resolveLocalHubStartCwd()
          });
        }
        return waitForLocalHub(candidates);
      } catch {
        return false;
      }
    })();
  }

  try {
    return await localHubStartInFlight;
  } finally {
    localHubStartInFlight = null;
  }
}

export function localHubBaseUrl(): string {
  const saved = localStorage.getItem(LOCAL_HUB_STORAGE_KEY);
  if (saved && saved.trim()) {
    return withHubUrlScheme(saved);
  }
  return withHubUrlScheme(DEFAULT_LOCAL_HUB_URL);
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

function cleanupLegacyLocalAuthArtifacts(): void {
  if (localLegacyCleanupDone) {
    return;
  }
  localLegacyCleanupDone = true;
  localStorage.removeItem(LEGACY_LOCAL_AUTO_PASSWORD_STORAGE_KEY);
  void deleteToken(LOCAL_TOKEN_PROFILE_ID).catch(() => undefined);
}

async function resolveLocalWorkspaceId(serverUrl: string): Promise<string> {
  const resp = await hubClient.listWorkspaces(serverUrl, "");
  return pickWorkspaceId(resp.workspaces);
}

async function resolveLocalHubContextInternal(allowAutoStart: boolean): Promise<HubContext> {
  cleanupLegacyLocalAuthArtifacts();

  const preferredServerUrl = localHubBaseUrl();
  const candidates = buildLocalHubCandidates(preferredServerUrl);

  for (const serverUrl of candidates) {
    try {
      await hubClient.getHealth(serverUrl);
      const workspaceId = await resolveLocalWorkspaceId(serverUrl);
      persistLocalHubBaseUrl(serverUrl);
      return { serverUrl, token: "", workspaceId };
    } catch {
      // continue to next candidate
    }
  }

  if (allowAutoStart) {
    const started = await ensureLocalHubProcessRunning(candidates);
    if (started) {
      return resolveLocalHubContextInternal(false);
    }
  }

  throw toError({
    code: "NETWORK_OR_RUNTIME_ERROR",
    message: "Local hub is unreachable. Please ensure Hub is running.",
    retryable: true
  });
}

export async function ensureLocalHubContext(): Promise<HubContext> {
  return resolveLocalHubContextInternal(true);
}

// Backward-compatible export name; local mode no longer performs auth/login.
export async function ensureLocalHubAuth(): Promise<HubContext> {
  return ensureLocalHubContext();
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
      const payload = await hubClient.getRuntimeHealth(ctx.serverUrl, ctx.token, ctx.workspaceId);
      return { ok: payload.runtime_status === "online" };
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
    return ensureLocalHubContext();
  }
  return resolveRemoteHubContext(profile);
}

export function getSessionDataSource(profile: WorkspaceProfile | undefined): SessionDataSource {
  if (!profile || profile.kind === "local") {
    return makeHubSessionDataSource("local", ensureLocalHubContext);
  }

  return makeHubSessionDataSource("remote", () => resolveRemoteHubContext(profile));
}
