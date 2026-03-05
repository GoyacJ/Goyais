import { defineStore } from "pinia";

import { resolveDiffCapability } from "@/modules/conversation/services";
import {
  buildConversationSnapshot
} from "@/modules/conversation/store/conversationSnapshots";
import { normalizeExecutionList } from "@/modules/conversation/store/executionMerge";
import { pinia } from "@/shared/stores/pinia";
import type {
  ChangeSetCapability,
  ConnectionStatus,
  Conversation,
  ConversationChangeSet,
  ConversationDetailResponse,
  ConversationMessage,
  ConversationMode,
  ConversationSnapshot,
  DiffItem,
  Execution,
  ExecutionEvent,
  InspectorTabKey,
  ProjectKind,
  Run,
  Session,
  SessionChangeSet,
  SessionDetailResponse,
  SessionSnapshot
} from "@/shared/types/api";

type SessionMode = ConversationMode;

export type StreamHandle = {
  close: () => void;
  lastEventId: () => string;
};

export type SessionRuntime = {
  messages: ConversationMessage[];
  events: ExecutionEvent[];
  runs: Run[];
  // Backward-compatibility projection while callers migrate to `runs`.
  executions: Run[];
  snapshots: ConversationSnapshot[];
  draft: string;
  mode: ConversationMode;
  modelId: string;
  ruleIds: string[];
  skillIds: string[];
  mcpIds: string[];
  status: ConnectionStatus;
  diff: DiffItem[];
  projectKind: ProjectKind;
  diffCapability: ChangeSetCapability;
  changeSet: SessionChangeSet | null;
  inspectorTab: InspectorTabKey;
  worktreeRef: string | null;
  hydrated: boolean;
  lastEventId: string;
  processedEventKeys: string[];
  processedEventKeySet: Set<string>;
  completionMessageKeys: string[];
  completionMessageKeySet: Set<string>;
};

export type ConversationRuntime = SessionRuntime;

export const MAX_RUNTIME_EVENTS = 1000;

type SessionState = {
  bySessionId: Record<string, SessionRuntime>;
  // Backward-compatibility projection while callers migrate to `bySessionId`.
  byConversationId: Record<string, SessionRuntime>;
  sessionTimers: Record<string, ReturnType<typeof setTimeout> | undefined>;
  timers: Record<string, ReturnType<typeof setTimeout> | undefined>;
  sessionStreams: Record<string, StreamHandle | undefined>;
  streams: Record<string, StreamHandle | undefined>;
  loading: boolean;
  error: string;
};

const initialState = (): SessionState => {
  const bySessionId: Record<string, SessionRuntime> = {};
  const sessionTimers: Record<string, ReturnType<typeof setTimeout> | undefined> = {};
  const sessionStreams: Record<string, StreamHandle | undefined> = {};
  return {
    bySessionId,
    byConversationId: bySessionId,
    sessionTimers,
    timers: sessionTimers,
    sessionStreams,
    streams: sessionStreams,
    loading: false,
    error: ""
  };
};

const useSessionStoreDefinition = defineStore("conversation", {
  state: (): SessionState => initialState()
});

export const useSessionStore = useSessionStoreDefinition;
export const sessionStore = useSessionStoreDefinition(pinia);
export const useConversationStore = useSessionStoreDefinition;
export const conversationStore = sessionStore;

export function resetSessionStore(): void {
  for (const timer of Object.values(sessionStore.sessionTimers)) {
    if (timer) {
      clearTimeout(timer);
    }
  }
  for (const stream of Object.values(sessionStore.sessionStreams)) {
    stream?.close();
  }

  const bySessionId: Record<string, SessionRuntime> = {};
  const sessionTimers: Record<string, ReturnType<typeof setTimeout> | undefined> = {};
  const sessionStreams: Record<string, StreamHandle | undefined> = {};
  sessionStore.bySessionId = bySessionId;
  sessionStore.byConversationId = bySessionId;
  sessionStore.sessionTimers = sessionTimers;
  sessionStore.timers = sessionTimers;
  sessionStore.sessionStreams = sessionStreams;
  sessionStore.streams = sessionStreams;
  sessionStore.loading = false;
  sessionStore.error = "";
}

export function resetConversationStore(): void {
  resetSessionStore();
}

export function ensureSessionRuntime(
  session: Session,
  isGitProject: boolean
): SessionRuntime {
  const existing = sessionStore.bySessionId[session.id];
  if (existing) {
    return ensureLegacyExecutionAlias(existing);
  }

  const runs: Run[] = [];
  const runtime: SessionRuntime = ensureLegacyExecutionAlias({
    messages: [],
    events: [],
    runs,
    executions: runs,
    snapshots: [],
    draft: "",
    mode: session.default_mode,
    modelId: session.model_config_id,
    ruleIds: [...(session.rule_ids ?? [])],
    skillIds: [...(session.skill_ids ?? [])],
    mcpIds: [...(session.mcp_ids ?? [])],
    status: "connected",
    diff: [],
    projectKind: isGitProject ? "git" : "non_git",
    diffCapability: resolveDiffCapability(isGitProject),
    changeSet: null,
    inspectorTab: "diff",
    worktreeRef: null,
    hydrated: false,
    lastEventId: "",
    processedEventKeys: [],
    processedEventKeySet: new Set<string>(),
    completionMessageKeys: [],
    completionMessageKeySet: new Set<string>()
  });

  sessionStore.bySessionId[session.id] = runtime;
  return runtime;
}

export function ensureConversationRuntime(
  conversation: Conversation,
  isGitProject: boolean
): ConversationRuntime {
  return ensureSessionRuntime(conversation, isGitProject);
}

export function hydrateSessionRuntime(
  session: Session,
  isGitProject: boolean,
  detail: SessionDetailResponse
): SessionRuntime {
  const runtime = ensureSessionRuntime(session, isGitProject);
  runtime.mode = detail.session.default_mode;
  runtime.modelId = detail.session.model_config_id;
  runtime.ruleIds = [...(detail.session.rule_ids ?? [])];
  runtime.skillIds = [...(detail.session.skill_ids ?? [])];
  runtime.mcpIds = [...(detail.session.mcp_ids ?? [])];
  runtime.projectKind = isGitProject ? "git" : "non_git";
  runtime.messages = detail.messages.map((message) => ({ ...message }));
  const runs = detail.runs.map((run) => ({
    ...run,
    model_snapshot: {
      ...run.model_snapshot
    },
    agent_config_snapshot: run.agent_config_snapshot
      ? { ...run.agent_config_snapshot }
      : undefined
  }));
  runtime.runs = runs;
  runtime.executions = runs;
  runtime.snapshots = detail.snapshots.map((snapshot) => ({
    ...snapshot,
    messages: snapshot.messages.map((message) => ({ ...message })),
    execution_snapshots: snapshot.execution_snapshots?.map((item) => ({ ...item })),
    execution_ids: [...snapshot.execution_ids]
  }));

  const latestSnapshot = runtime.snapshots[runtime.snapshots.length - 1];
  runtime.worktreeRef = latestSnapshot?.worktree_ref ?? null;
  runtime.inspectorTab = latestSnapshot?.inspector_state.tab ?? "diff";
  runtime.diff = [];
  runtime.changeSet = null;
  runtime.diffCapability = resolveDiffCapability(isGitProject);
  runtime.hydrated = true;
  return ensureLegacyExecutionAlias(runtime);
}

export function hydrateConversationRuntime(
  conversation: Conversation,
  isGitProject: boolean,
  detail: ConversationDetailResponse
): ConversationRuntime {
  return hydrateSessionRuntime(conversation, isGitProject, toSessionDetailResponse(detail));
}

export function setSessionChangeSet(sessionId: string, changeSet: SessionChangeSet | null): void {
  const runtime = sessionStore.bySessionId[sessionId];
  if (!runtime) {
    return;
  }

  runtime.changeSet = changeSet;
  if (!changeSet) {
    runtime.diff = [];
    runtime.diffCapability = resolveDiffCapability(runtime.projectKind === "git");
    return;
  }

  runtime.projectKind = changeSet.project_kind;
  runtime.diff = changeSet.entries.map((entry) => ({
    id: entry.entry_id,
    path: entry.path,
    change_type: entry.change_type,
    summary: entry.summary,
    added_lines: entry.added_lines,
    deleted_lines: entry.deleted_lines
  }));
  runtime.diffCapability = {
    can_commit: changeSet.capability.can_commit,
    can_discard: changeSet.capability.can_discard,
    can_export: changeSet.capability.can_export,
    can_export_patch: changeSet.capability.can_export_patch ?? changeSet.capability.can_export,
    reason: changeSet.capability.reason
  };
}

export function setConversationChangeSet(conversationId: string, changeSet: ConversationChangeSet | null): void {
  setSessionChangeSet(conversationId, changeSet);
}

export function getSessionRuntime(sessionId: string): SessionRuntime | undefined {
  const runtime = sessionStore.bySessionId[sessionId];
  if (!runtime) {
    return undefined;
  }
  return ensureLegacyExecutionAlias(runtime);
}

export function getConversationRuntime(conversationId: string): ConversationRuntime | undefined {
  return getSessionRuntime(conversationId);
}

export function appendRuntimeEvent(runtime: SessionRuntime, event: ExecutionEvent): void {
  const eventID = event.event_id?.trim();
  if (eventID) {
    runtime.lastEventId = eventID;
  }
  runtime.events.push(event);
  if (runtime.events.length > MAX_RUNTIME_EVENTS) {
    runtime.events.splice(0, runtime.events.length - MAX_RUNTIME_EVENTS);
  }
}

export function setSessionDraft(sessionId: string, draft: string): void {
  const runtime = sessionStore.bySessionId[sessionId];
  if (runtime) {
    runtime.draft = draft;
  }
}

export function setConversationDraft(conversationId: string, draft: string): void {
  setSessionDraft(conversationId, draft);
}

export function setSessionMode(sessionId: string, mode: SessionMode): void {
  const runtime = sessionStore.bySessionId[sessionId];
  if (runtime) {
    runtime.mode = mode;
  }
}

export function setConversationMode(conversationId: string, mode: ConversationMode): void {
  setSessionMode(conversationId, mode);
}

export function setSessionModel(sessionId: string, modelId: string): void {
  const runtime = sessionStore.bySessionId[sessionId];
  if (runtime) {
    runtime.modelId = modelId;
  }
}

export function setConversationModel(conversationId: string, modelId: string): void {
  setSessionModel(conversationId, modelId);
}

export function setSessionInspectorTab(sessionId: string, tab: InspectorTabKey): void {
  const runtime = sessionStore.bySessionId[sessionId];
  if (runtime) {
    runtime.inspectorTab = tab;
  }
}

export function setConversationInspectorTab(conversationId: string, tab: InspectorTabKey): void {
  setSessionInspectorTab(conversationId, tab);
}

export function setConversationError(error: string): void {
  sessionStore.error = error;
}

export function clearSessionTimer(sessionId: string): void {
  const timer = sessionStore.sessionTimers[sessionId];
  if (timer) {
    clearTimeout(timer);
  }
  delete sessionStore.sessionTimers[sessionId];
}

export function clearConversationTimer(conversationId: string): void {
  clearSessionTimer(conversationId);
}

export function createSessionSnapshot(runtime: SessionRuntime, sessionId: string, rollbackPointMessageId: string): SessionSnapshot {
  return buildConversationSnapshot(runtime, sessionId, rollbackPointMessageId);
}

export function createConversationSnapshot(runtime: ConversationRuntime, conversationId: string, rollbackPointMessageId: string): ConversationSnapshot {
  return createSessionSnapshot(runtime, conversationId, rollbackPointMessageId);
}

export function pushSessionSnapshot(sessionId: string, snapshot: SessionSnapshot): void {
  const runtime = sessionStore.bySessionId[sessionId];
  if (!runtime) {
    return;
  }

  runtime.snapshots.push(snapshot);
}

export function pushConversationSnapshot(conversationId: string, snapshot: ConversationSnapshot): void {
  pushSessionSnapshot(conversationId, snapshot);
}

export function findSessionSnapshotForMessage(sessionId: string, messageId: string): SessionSnapshot | undefined {
  const runtime = sessionStore.bySessionId[sessionId];
  if (!runtime) {
    return undefined;
  }

  return [...runtime.snapshots].reverse().find((snapshot) => snapshot.rollback_point_message_id === messageId);
}

export function findSnapshotForMessage(conversationId: string, messageId: string): ConversationSnapshot | undefined {
  return findSessionSnapshotForMessage(conversationId, messageId);
}

export function countActiveAndQueued(runtime: SessionRuntime): number {
  const runs = normalizeExecutionList(runtime.executions);
  return runs.filter((run) =>
    run.state === "queued" ||
    run.state === "pending" ||
    run.state === "executing" ||
    run.state === "confirming" ||
    run.state === "awaiting_input"
  ).length;
}

export function getRunStateCounts(runtime: SessionRuntime): {
  queued: number;
  pending: number;
  executing: number;
} {
  const runs = normalizeExecutionList(runtime.executions);
  return runs.reduce(
    (acc, run) => {
      if (run.state === "queued") {
        acc.queued += 1;
      } else if (run.state === "pending") {
        acc.pending += 1;
      } else if (run.state === "executing" || run.state === "confirming" || run.state === "awaiting_input") {
        acc.executing += 1;
      }
      return acc;
    },
    { queued: 0, pending: 0, executing: 0 }
  );
}

export function hasUnfinishedRuns(runtime: SessionRuntime): boolean {
  const counts = getRunStateCounts(runtime);
  return counts.queued > 0 || counts.pending > 0 || counts.executing > 0;
}

export function hasUnfinishedExecutions(runtime: ConversationRuntime): boolean {
  return hasUnfinishedRuns(runtime);
}

export function getLatestFinishedRun(sessionId: string): Execution | undefined {
  const runtime = sessionStore.bySessionId[sessionId];
  if (!runtime) {
    return undefined;
  }

  const runs = normalizeExecutionList(runtime.executions);
  return [...runs]
    .reverse()
    .find((run) => run.state === "completed" || run.state === "failed" || run.state === "cancelled");
}

export function getLatestFinishedExecution(conversationId: string): Execution | undefined {
  return getLatestFinishedRun(conversationId);
}

function toSessionDetailResponse(detail: ConversationDetailResponse): SessionDetailResponse {
  const session = detail.session ?? detail.conversation;
  const runs = detail.runs ?? detail.executions;
  return {
    session,
    messages: detail.messages,
    runs,
    snapshots: detail.snapshots,
    conversation: detail.conversation,
    executions: detail.executions
  };
}

function ensureLegacyExecutionAlias(runtime: SessionRuntime): SessionRuntime {
  runtime.executions = runtime.runs;
  return runtime;
}
