import { normalizeExecutionList } from "@/modules/conversation/store/executionMerge";
import { createMockId } from "@/shared/services/mockData";
import type {
  ConversationMessage,
  ConversationSnapshot,
  QueueState
} from "@/shared/types/api";

import type { ConversationRuntime } from "@/modules/conversation/store/state";

export function createInitialMessages(conversationId: string): ConversationMessage[] {
  void conversationId;
  return [];
}

export function buildConversationSnapshot(
  runtime: ConversationRuntime,
  conversationId: string,
  rollbackPointMessageId: string
): ConversationSnapshot {
  const executions = normalizeExecutionList(runtime.executions);
  const executionSnapshots = executions.map((execution) => ({
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
    execution_ids: executions.map((execution) => execution.id),
    created_at: new Date().toISOString()
  };
}

function deriveQueueState(runtime: ConversationRuntime): QueueState {
  const executions = normalizeExecutionList(runtime.executions);
  const hasRunning = executions.some((execution) =>
    execution.state === "pending" || execution.state === "executing"
  );
  const hasQueued = executions.some((execution) => execution.state === "queued");
  if (hasRunning) {
    return "running";
  }
  if (hasQueued) {
    return "queued";
  }
  return "idle";
}
