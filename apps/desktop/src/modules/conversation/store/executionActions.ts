import {
  cancelExecution,
  commitExecution,
  createExecution,
  discardExecution,
  loadExecutionDiff,
  rollbackExecution
} from "@/modules/conversation/services";
import { createExecutionEvent } from "@/modules/conversation/store/events";
import {
  applyExecutionState,
  dedupeExecutions,
  ensureExecution,
  parseDiff,
  restoreExecutionsFromSnapshot,
  upsertExecutionFromServer
} from "@/modules/conversation/store/executionRuntime";
import {
  appendRuntimeEvent,
  conversationStore,
  createConversationSnapshot,
  ensureConversationRuntime,
  findSnapshotForMessage,
  getLatestFinishedExecution,
  pushConversationSnapshot
} from "@/modules/conversation/store/state";
import { toDisplayError } from "@/shared/services/errorMapper";
import { createMockId } from "@/shared/services/mockData";
import type { Conversation, ConversationMessage, ExecutionEvent } from "@/shared/types/api";

export async function submitConversationMessage(
  conversation: Conversation,
  isGitProject: boolean
): Promise<void> {
  const runtime = ensureConversationRuntime(conversation, isGitProject);
  const content = runtime.draft.trim();
  if (content === "") {
    return;
  }

  runtime.draft = "";
  const queueIndex = runtime.executions.length;

  const userMessage: ConversationMessage = {
    id: createMockId("msg"),
    conversation_id: conversation.id,
    role: "user",
    content,
    queue_index: queueIndex,
    can_rollback: true,
    created_at: new Date().toISOString()
  };
  runtime.messages.push(userMessage);
  pushConversationSnapshot(
    conversation.id,
    createConversationSnapshot(runtime, conversation.id, userMessage.id)
  );

  try {
    const response = await createExecution(conversation, {
      content,
      mode: runtime.mode,
      model_id: runtime.modelId
    });

    upsertExecutionFromServer(runtime, response.execution);
    dedupeExecutions(runtime);
    appendRuntimeEvent(
      runtime,
      createExecutionEvent(conversation.id, response.execution.id, response.queue_index, "message_received", {
        message_id: response.execution.message_id,
        queue_state: response.queue_state
      })
    );
  } catch (error) {
    conversationStore.error = toDisplayError(error);
    runtime.messages.push({
      id: createMockId("msg"),
      conversation_id: conversation.id,
      role: "system",
      content: conversationStore.error,
      created_at: new Date().toISOString()
    });
  }
}

export async function stopConversationExecution(conversation: Conversation): Promise<void> {
  const runtime = conversationStore.byConversationId[conversation.id];
  if (!runtime) {
    return;
  }

  const active = runtime.executions.find((item) => item.state === "executing" || item.state === "pending");
  if (!active) {
    return;
  }

  try {
    await cancelExecution(conversation.id, active.id);
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

export async function rollbackConversationToMessage(conversationId: string, messageId: string): Promise<void> {
  const runtime = conversationStore.byConversationId[conversationId];
  if (!runtime) {
    return;
  }

  const targetMessage = runtime.messages.find((message) => message.id === messageId);
  if (!targetMessage || targetMessage.role !== "user") {
    return;
  }

  appendRuntimeEvent(
    runtime,
    createExecutionEvent(conversationId, "", targetMessage.queue_index ?? 0, "thinking_delta", {
      stage: "rollback_requested",
      message_id: messageId
    })
  );

  try {
    await rollbackExecution(conversationId, messageId);
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }

  const snapshot = findSnapshotForMessage(conversationId, messageId);
  if (!snapshot) {
    return;
  }

  runtime.messages = snapshot.messages.map((message) => ({ ...message }));
  runtime.executions = restoreExecutionsFromSnapshot(runtime, conversationId, snapshot);
  runtime.snapshots = runtime.snapshots.filter((item) => item.created_at <= snapshot.created_at);
  runtime.worktreeRef = snapshot.worktree_ref;
  runtime.inspectorTab = snapshot.inspector_state.tab;
  runtime.diff = [];

  appendRuntimeEvent(
    runtime,
    createExecutionEvent(conversationId, "", targetMessage.queue_index ?? 0, "thinking_delta", {
      stage: "snapshot_applied",
      message_id: messageId
    })
  );

  appendRuntimeEvent(
    runtime,
    createExecutionEvent(conversationId, "", targetMessage.queue_index ?? 0, "thinking_delta", {
      stage: "rollback_completed",
      message_id: messageId
    })
  );
}

export async function commitLatestDiff(conversationId: string): Promise<void> {
  const execution = getLatestFinishedExecution(conversationId);
  if (!execution) {
    return;
  }

  try {
    await commitExecution(execution.id);
    const runtime = conversationStore.byConversationId[conversationId];
    if (runtime) {
      runtime.diff = [];
    }
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

export async function discardLatestDiff(conversationId: string): Promise<void> {
  const execution = getLatestFinishedExecution(conversationId);
  if (!execution) {
    return;
  }

  try {
    await discardExecution(execution.id);
    const runtime = conversationStore.byConversationId[conversationId];
    if (runtime) {
      runtime.diff = [];
    }
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

export async function refreshExecutionDiff(conversationId: string, executionId: string): Promise<void> {
  const runtime = conversationStore.byConversationId[conversationId];
  if (!runtime) {
    return;
  }
  try {
    runtime.diff = await loadExecutionDiff(executionId);
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

export function applyIncomingExecutionEvent(conversationId: string, event: ExecutionEvent): void {
  const runtime = conversationStore.byConversationId[conversationId];
  if (!runtime) {
    return;
  }

  appendRuntimeEvent(runtime, event);

  if (event.execution_id) {
    const execution = ensureExecution(runtime, conversationId, event);
    applyExecutionState(execution, event);
    dedupeExecutions(runtime);
  }

  if (event.type === "diff_generated") {
    runtime.diff = parseDiff(event.payload);
  }

  if (event.type === "execution_done") {
    const content = typeof event.payload.content === "string" && event.payload.content.trim() !== ""
      ? event.payload.content
      : `Execution ${event.execution_id} completed.`;
    runtime.messages.push({
      id: createMockId("msg"),
      conversation_id: conversationId,
      role: "assistant",
      content,
      created_at: new Date().toISOString()
    });
  }

  if (event.type === "execution_error") {
    const content = typeof event.payload.message === "string" && event.payload.message.trim() !== ""
      ? event.payload.message
      : "Execution failed.";
    runtime.messages.push({
      id: createMockId("msg"),
      conversation_id: conversationId,
      role: "system",
      content,
      created_at: new Date().toISOString()
    });
  }
}
