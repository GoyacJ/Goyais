import { reactive } from "vue";

import { resolveDiffCapability } from "@/modules/conversation/services";
import {
  buildConversationSnapshot,
  createInitialMessages
} from "@/modules/conversation/store/conversationSnapshots";
import { normalizeExecutionList } from "@/modules/conversation/store/executionMerge";
import type {
  ConversationDetailResponse,
  ConnectionStatus,
  Conversation,
  ConversationMessage,
  ConversationMode,
  ConversationSnapshot,
  DiffCapability,
  DiffItem,
  Execution,
  ExecutionEvent,
  InspectorTabKey
} from "@/shared/types/api";

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
  status: ConnectionStatus;
  diff: DiffItem[];
  diffCapability: DiffCapability;
  inspectorTab: InspectorTabKey;
  worktreeRef: string | null;
  hydrated: boolean;
  lastEventId: string;
  processedEventKeys: string[];
  processedEventKeySet: Set<string>;
  completionMessageKeys: string[];
  completionMessageKeySet: Set<string>;
};

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

export const conversationStore = reactive<ConversationState>({ ...initialState });

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
    modelId: conversation.model_id,
    status: "connected",
    diff: [],
    diffCapability: resolveDiffCapability(isGitProject),
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

export function hydrateConversationRuntime(
  conversation: Conversation,
  isGitProject: boolean,
  detail: ConversationDetailResponse
): ConversationRuntime {
  const runtime = ensureConversationRuntime(conversation, isGitProject);
  runtime.mode = detail.conversation.default_mode;
  runtime.modelId = detail.conversation.model_id;
  runtime.messages = detail.messages.length > 0 ? detail.messages.map((message) => ({ ...message })) : createInitialMessages(conversation.id);
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
  runtime.hydrated = true;
  return runtime;
}

export function getConversationRuntime(conversationId: string): ConversationRuntime | undefined {
  return conversationStore.byConversationId[conversationId];
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

export function setConversationMode(conversationId: string, mode: ConversationMode): void {
  const runtime = conversationStore.byConversationId[conversationId];
  if (runtime) {
    runtime.mode = mode;
  }
}

export function setConversationModel(conversationId: string, modelId: string): void {
  const runtime = conversationStore.byConversationId[conversationId];
  if (runtime) {
    runtime.modelId = modelId;
  }
}

export function setConversationInspectorTab(conversationId: string, tab: InspectorTabKey): void {
  const runtime = conversationStore.byConversationId[conversationId];
  if (runtime) {
    runtime.inspectorTab = tab;
  }
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

export function createConversationSnapshot(runtime: ConversationRuntime, conversationId: string, rollbackPointMessageId: string): ConversationSnapshot {
  return buildConversationSnapshot(runtime, conversationId, rollbackPointMessageId);
}

export function pushConversationSnapshot(conversationId: string, snapshot: ConversationSnapshot): void {
  const runtime = conversationStore.byConversationId[conversationId];
  if (!runtime) {
    return;
  }

  runtime.snapshots.push(snapshot);
}

export function findSnapshotForMessage(conversationId: string, messageId: string): ConversationSnapshot | undefined {
  const runtime = conversationStore.byConversationId[conversationId];
  if (!runtime) {
    return undefined;
  }

  return [...runtime.snapshots].reverse().find((snapshot) => snapshot.rollback_point_message_id === messageId);
}

export function countActiveAndQueued(runtime: ConversationRuntime): number {
  const executions = normalizeExecutionList(runtime.executions);
  return executions.filter((execution) =>
    execution.state === "queued" || execution.state === "pending" || execution.state === "executing"
  ).length;
}

export function getExecutionStateCounts(runtime: ConversationRuntime): {
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
      } else if (execution.state === "executing") {
        acc.executing += 1;
      }
      return acc;
    },
    { queued: 0, pending: 0, executing: 0 }
  );
}

export function hasUnfinishedExecutions(runtime: ConversationRuntime): boolean {
  const counts = getExecutionStateCounts(runtime);
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
