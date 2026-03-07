import { normalizeExecutionList } from "@/modules/session/store/executionMerge";
import { createMockId } from "@/shared/utils/id";
import type {
  SessionMessage,
  SessionSnapshot,
  QueueState
} from "@/shared/types/api";

import type { SessionRuntime } from "@/modules/session/store/state";

export function createInitialMessages(conversationId: string): SessionMessage[] {
  void conversationId;
  return [];
}

export function buildConversationSnapshot(
  runtime: SessionRuntime,
  conversationId: string,
  rollbackPointMessageId: string
): SessionSnapshot {
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
    session_id: conversationId,
    rollback_point_message_id: rollbackPointMessageId,
    queue_state: deriveQueueState(runtime),
    worktree_ref: runtime.worktreeRef,
    inspector_state: {
      tab: runtime.inspectorTab
    },
    messages: runtime.messages.map((message) => ({ ...message })),
    execution_snapshots: executionSnapshots,
    run_ids: executions.map((execution) => execution.id),
    created_at: new Date().toISOString()
  };
}

function deriveQueueState(runtime: SessionRuntime): QueueState {
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
