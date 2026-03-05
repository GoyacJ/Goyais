import { defineStore } from "pinia";

import { resolveDiffCapability } from "@/modules/conversation/services";
import {
  buildConversationSnapshot
} from "@/modules/conversation/store/conversationSnapshots";
import { normalizeExecutionList } from "@/modules/conversation/store/executionMerge";
import { pinia } from "@/shared/stores/pinia";
import type {
  ChangeSetCapability,
  ConversationDetailResponse,
  ConversationChangeSet,
  ConnectionStatus,
  Conversation,
  ConversationMessage,
  ConversationMode,
  ConversationSnapshot,
  DiffItem,
  Execution,
  ExecutionEvent,
  InspectorTabKey,
  ProjectKind,
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

export type ConversationRuntime = {
  messages: ConversationMessage[];
  events: ExecutionEvent[];
  executions: Execution[];
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
  changeSet: ConversationChangeSet | null;
  inspectorTab: InspectorTabKey;
  worktreeRef: string | null;
  hydrated: boolean;
  lastEventId: string;
  processedEventKeys: string[];
  processedEventKeySet: Set<string>;
  completionMessageKeys: string[];
  completionMessageKeySet: Set<string>;
};

export type SessionRuntime = ConversationRuntime;

export const MAX_RUNTIME_EVENTS = 1000;

type ConversationState = {
  byConversationId: Record<string, ConversationRuntime>;
  timers: Record<string, ReturnType<typeof setTimeout> | undefined>;
  streams: Record<string, StreamHandle | undefined>;
  loading: boolean;
  error: string;
};

const initialState: ConversationState = {
  byConversationId: {},
  timers: {},
  streams: {},
  loading: false,
  error: ""
};

const useConversationStoreDefinition = defineStore("conversation", {
  state: (): ConversationState => ({ ...initialState })
});

export const useConversationStore = useConversationStoreDefinition;
export const conversationStore = useConversationStoreDefinition(pinia);
export const useSessionStore = useConversationStoreDefinition;
export const sessionStore = conversationStore;

export function resetConversationStore(): void {
  for (const timer of Object.values(conversationStore.timers)) {
    if (timer) {
      clearTimeout(timer);
    }
  }
  for (const stream of Object.values(conversationStore.streams)) {
    stream?.close();
  }

  conversationStore.byConversationId = {};
  conversationStore.timers = {};
  conversationStore.streams = {};
  conversationStore.loading = false;
  conversationStore.error = "";
}

export function ensureConversationRuntime(
  conversation: Conversation,
  isGitProject: boolean
): ConversationRuntime {
  const existing = conversationStore.byConversationId[conversation.id];
  if (existing) {
    return existing;
  }

  const runtime: ConversationRuntime = {
    messages: [],
    events: [],
    executions: [],
    snapshots: [],
    draft: "",
    mode: conversation.default_mode,
    modelId: conversation.model_config_id,
    ruleIds: [...(conversation.rule_ids ?? [])],
    skillIds: [...(conversation.skill_ids ?? [])],
    mcpIds: [...(conversation.mcp_ids ?? [])],
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
  };

  conversationStore.byConversationId[conversation.id] = runtime;
  return runtime;
}

export function ensureSessionRuntime(
  session: Session,
  isGitProject: boolean
): SessionRuntime {
  return ensureConversationRuntime(session, isGitProject);
}

export function hydrateConversationRuntime(
  conversation: Conversation,
  isGitProject: boolean,
  detail: ConversationDetailResponse
): ConversationRuntime {
  const runtime = ensureConversationRuntime(conversation, isGitProject);
  runtime.mode = detail.conversation.default_mode;
  runtime.modelId = detail.conversation.model_config_id;
  runtime.ruleIds = [...(detail.conversation.rule_ids ?? [])];
  runtime.skillIds = [...(detail.conversation.skill_ids ?? [])];
  runtime.mcpIds = [...(detail.conversation.mcp_ids ?? [])];
  runtime.projectKind = isGitProject ? "git" : "non_git";
  runtime.messages = detail.messages.map((message) => ({ ...message }));
  runtime.executions = detail.executions.map((execution) => ({
    ...execution,
    model_snapshot: {
      ...execution.model_snapshot
    },
    agent_config_snapshot: execution.agent_config_snapshot
      ? { ...execution.agent_config_snapshot }
      : undefined
  }));
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
  return runtime;
}

export function hydrateSessionRuntime(
  session: Session,
  isGitProject: boolean,
  detail: SessionDetailResponse
): SessionRuntime {
  return hydrateConversationRuntime(session, isGitProject, detail);
}

export function setConversationChangeSet(conversationId: string, changeSet: ConversationChangeSet | null): void {
  const runtime = conversationStore.byConversationId[conversationId];
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

export function setSessionChangeSet(sessionId: string, changeSet: SessionChangeSet | null): void {
  setConversationChangeSet(sessionId, changeSet);
}

export function getConversationRuntime(conversationId: string): ConversationRuntime | undefined {
  return conversationStore.byConversationId[conversationId];
}

export function getSessionRuntime(sessionId: string): SessionRuntime | undefined {
  return getConversationRuntime(sessionId);
}

export function appendRuntimeEvent(runtime: ConversationRuntime, event: ExecutionEvent): void {
  const eventID = event.event_id?.trim();
  if (eventID) {
    runtime.lastEventId = eventID;
  }
  runtime.events.push(event);
  if (runtime.events.length > MAX_RUNTIME_EVENTS) {
    runtime.events.splice(0, runtime.events.length - MAX_RUNTIME_EVENTS);
  }
}

export function setConversationDraft(conversationId: string, draft: string): void {
  const runtime = conversationStore.byConversationId[conversationId];
  if (runtime) {
    runtime.draft = draft;
  }
}

export function setSessionDraft(sessionId: string, draft: string): void {
  setConversationDraft(sessionId, draft);
}

export function setConversationMode(conversationId: string, mode: ConversationMode): void {
  const runtime = conversationStore.byConversationId[conversationId];
  if (runtime) {
    runtime.mode = mode;
  }
}

export function setSessionMode(sessionId: string, mode: SessionMode): void {
  setConversationMode(sessionId, mode);
}

export function setConversationModel(conversationId: string, modelId: string): void {
  const runtime = conversationStore.byConversationId[conversationId];
  if (runtime) {
    runtime.modelId = modelId;
  }
}

export function setSessionModel(sessionId: string, modelId: string): void {
  setConversationModel(sessionId, modelId);
}

export function setConversationInspectorTab(conversationId: string, tab: InspectorTabKey): void {
  const runtime = conversationStore.byConversationId[conversationId];
  if (runtime) {
    runtime.inspectorTab = tab;
  }
}

export function setSessionInspectorTab(sessionId: string, tab: InspectorTabKey): void {
  setConversationInspectorTab(sessionId, tab);
}

export function setConversationError(error: string): void {
  conversationStore.error = error;
}

export function clearConversationTimer(conversationId: string): void {
  const timer = conversationStore.timers[conversationId];
  if (timer) {
    clearTimeout(timer);
  }
  delete conversationStore.timers[conversationId];
}

export function clearSessionTimer(sessionId: string): void {
  clearConversationTimer(sessionId);
}

export function createConversationSnapshot(runtime: ConversationRuntime, conversationId: string, rollbackPointMessageId: string): ConversationSnapshot {
  return buildConversationSnapshot(runtime, conversationId, rollbackPointMessageId);
}

export function createSessionSnapshot(runtime: SessionRuntime, sessionId: string, rollbackPointMessageId: string): SessionSnapshot {
  return createConversationSnapshot(runtime, sessionId, rollbackPointMessageId);
}

export function pushConversationSnapshot(conversationId: string, snapshot: ConversationSnapshot): void {
  const runtime = conversationStore.byConversationId[conversationId];
  if (!runtime) {
    return;
  }

  runtime.snapshots.push(snapshot);
}

export function pushSessionSnapshot(sessionId: string, snapshot: SessionSnapshot): void {
  pushConversationSnapshot(sessionId, snapshot);
}

export function findSnapshotForMessage(conversationId: string, messageId: string): ConversationSnapshot | undefined {
  const runtime = conversationStore.byConversationId[conversationId];
  if (!runtime) {
    return undefined;
  }

  return [...runtime.snapshots].reverse().find((snapshot) => snapshot.rollback_point_message_id === messageId);
}

export function findSessionSnapshotForMessage(sessionId: string, messageId: string): SessionSnapshot | undefined {
  return findSnapshotForMessage(sessionId, messageId);
}

export function countActiveAndQueued(runtime: ConversationRuntime): number {
  const executions = normalizeExecutionList(runtime.executions);
  return executions.filter((execution) =>
    execution.state === "queued" ||
    execution.state === "pending" ||
    execution.state === "executing" ||
    execution.state === "confirming" ||
    execution.state === "awaiting_input"
  ).length;
}

export function getRunStateCounts(runtime: ConversationRuntime): {
  queued: number;
  pending: number;
  executing: number;
} {
  const executions = normalizeExecutionList(runtime.executions);
  return executions.reduce(
    (acc, execution) => {
      if (execution.state === "queued") {
        acc.queued += 1;
      } else if (execution.state === "pending") {
        acc.pending += 1;
      } else if (execution.state === "executing" || execution.state === "confirming" || execution.state === "awaiting_input") {
        acc.executing += 1;
      }
      return acc;
    },
    { queued: 0, pending: 0, executing: 0 }
  );
}

export function hasUnfinishedExecutions(runtime: ConversationRuntime): boolean {
  const counts = getRunStateCounts(runtime);
  return counts.queued > 0 || counts.pending > 0 || counts.executing > 0;
}

export function getLatestFinishedExecution(conversationId: string): Execution | undefined {
  const runtime = conversationStore.byConversationId[conversationId];
  if (!runtime) {
    return undefined;
  }

  const executions = normalizeExecutionList(runtime.executions);
  return [...executions]
    .reverse()
    .find((execution) => execution.state === "completed" || execution.state === "failed" || execution.state === "cancelled");
}

export function getLatestFinishedRun(sessionId: string): Execution | undefined {
  return getLatestFinishedExecution(sessionId);
}
