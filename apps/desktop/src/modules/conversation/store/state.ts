import { reactive } from "vue";

import { resolveDiffCapability } from "@/modules/conversation/services";
import { createMockId } from "@/shared/services/mockData";
import type {
  ConnectionStatus,
  Conversation,
  ConversationMessage,
  ConversationMode,
  ConversationSnapshot,
  DiffCapability,
  DiffItem,
  Execution,
  ExecutionEvent,
  InspectorTabKey,
  QueueState
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
};

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
    messages: createInitialMessages(conversation.id),
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
    worktreeRef: null
  };

  conversationStore.byConversationId[conversation.id] = runtime;
  return runtime;
}

export function getConversationRuntime(conversationId: string): ConversationRuntime | undefined {
  return conversationStore.byConversationId[conversationId];
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

export function createInitialMessages(conversationId: string): ConversationMessage[] {
  return [
    {
      id: createMockId("msg"),
      conversation_id: conversationId,
      role: "assistant",
      content: "欢迎使用 Goyais，当前会话已准备就绪。",
      created_at: new Date().toISOString()
    }
  ];
}

export function deriveQueueState(runtime: ConversationRuntime): QueueState {
  const hasRunning = runtime.executions.some((execution) => execution.state === "executing" || execution.state === "confirming");
  const hasQueued = runtime.executions.some((execution) => execution.state === "queued");
  if (hasRunning) {
    return "running";
  }
  if (hasQueued) {
    return "queued";
  }
  return "idle";
}

export function createConversationSnapshot(runtime: ConversationRuntime, conversationId: string, rollbackPointMessageId: string): ConversationSnapshot {
  const executionSnapshots = runtime.executions.map((execution) => ({
    id: execution.id,
    state: execution.state,
    queue_index: execution.queue_index,
    message_id: execution.message_id,
    updated_at: execution.updated_at
  }));

  return {
    id: createMockId("snap"),
    conversation_id: conversationId,
    rollback_point_message_id: rollbackPointMessageId,
    queue_state: deriveQueueState(runtime),
    worktree_ref: runtime.worktreeRef,
    inspector_state: {
      tab: runtime.inspectorTab
    },
    messages: runtime.messages.map((message) => ({ ...message })),
    execution_snapshots: executionSnapshots,
    execution_ids: runtime.executions.map((execution) => execution.id),
    created_at: new Date().toISOString()
  };
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
  return runtime.executions.filter((execution) => execution.state === "queued" || execution.state === "executing").length;
}

export function getLatestFinishedExecution(conversationId: string): Execution | undefined {
  const runtime = conversationStore.byConversationId[conversationId];
  if (!runtime) {
    return undefined;
  }

  return [...runtime.executions]
    .reverse()
    .find((execution) => execution.state === "completed" || execution.state === "failed" || execution.state === "cancelled");
}
