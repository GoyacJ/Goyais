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
  buildEventDedupKey,
  rememberProcessedEvent,
  shouldAppendTerminalMessage
} from "@/modules/conversation/store/executionEventIdempotency";
import {
  appendRuntimeEvent,
  conversationStore,
  createConversationSnapshot,
  ensureConversationRuntime,
  findSnapshotForMessage,
  getLatestFinishedExecution,
  pushConversationSnapshot
} from "@/modules/conversation/store/state";
import type { ConversationRuntime } from "@/modules/conversation/store/state";
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

  const eventDedupKey = buildEventDedupKey(event);
  if (eventDedupKey !== "" && runtime.processedEventKeySet.has(eventDedupKey)) {
    return;
  }
  rememberProcessedEvent(runtime, eventDedupKey);

  appendRuntimeEvent(runtime, event);

  let previousState: string | undefined;
  let nextState: string | undefined;
  let messageID = "";
  if (event.execution_id) {
    previousState = runtime.executions.find((item) => item.id === event.execution_id)?.state;
    const execution = ensureExecution(runtime, conversationId, event);
    messageID = execution.message_id;
    applyExecutionState(execution, event);
    dedupeExecutions(runtime);
    nextState = runtime.executions.find((item) => item.id === event.execution_id)?.state;
  }

  if (event.type === "diff_generated") {
    runtime.diff = parseDiff(event.payload);
  }

  if (
    event.type === "execution_done" &&
    shouldAppendTerminalMessage(runtime, event, previousState, nextState, messageID, "assistant")
  ) {
    const content = typeof event.payload.content === "string" && event.payload.content.trim() !== ""
      ? event.payload.content
      : `Execution ${event.execution_id} completed.`;
    appendTerminalMessage(runtime, {
      id: createMockId("msg"),
      conversation_id: conversationId,
      role: "assistant",
      content,
      queue_index: event.queue_index,
      created_at: new Date().toISOString()
    });
  }

  if (
    event.type === "execution_error" &&
    shouldAppendTerminalMessage(runtime, event, previousState, nextState, messageID, "system")
  ) {
    const content = typeof event.payload.message === "string" && event.payload.message.trim() !== ""
      ? event.payload.message
      : "Execution failed.";
    appendTerminalMessage(runtime, {
      id: createMockId("msg"),
      conversation_id: conversationId,
      role: "system",
      content,
      queue_index: event.queue_index,
      created_at: new Date().toISOString()
    });
  }
}

function appendTerminalMessage(runtime: ConversationRuntime, message: ConversationMessage): void {
  if (typeof message.queue_index !== "number") {
    runtime.messages.push(message);
    return;
  }

  let insertAfter = -1;
  for (let index = 0; index < runtime.messages.length; index += 1) {
    const current = runtime.messages[index];
    if (typeof current.queue_index !== "number") {
      continue;
    }
    if (current.queue_index <= message.queue_index) {
      insertAfter = index;
    }
  }

  if (insertAfter < 0) {
    runtime.messages.push(message);
    return;
  }
  runtime.messages.splice(insertAfter + 1, 0, message);
}
