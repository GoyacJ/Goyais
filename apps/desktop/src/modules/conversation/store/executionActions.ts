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
  clearConversationTimer,
  conversationStore,
  countActiveAndQueued,
  createConversationSnapshot,
  ensureConversationRuntime,
  findSnapshotForMessage,
  getLatestFinishedExecution,
  pushConversationSnapshot,
  type ConversationRuntime
} from "@/modules/conversation/store/state";
import { toDisplayError } from "@/shared/services/errorMapper";
import { createMockId } from "@/shared/services/mockData";
import type { Conversation, ConversationMessage, Execution } from "@/shared/types/api";

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
  const queueIndex = countActiveAndQueued(runtime);

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

    const execution = {
      ...response.execution,
      message_id: userMessage.id,
      queue_index: queueIndex
    };

    runtime.executions.push(execution);
    runtime.events.push(
      createExecutionEvent(conversation.id, execution.id, queueIndex, "message_received", {
        message_id: userMessage.id
      })
    );
    startOrQueueExecution(conversation.id, runtime, execution);
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

  const active = runtime.executions.find((item) => item.state === "executing" || item.state === "confirming");
  if (!active) {
    return;
  }

  try {
    await cancelExecution(conversation.id, active.id);
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }

  clearConversationTimer(conversation.id);
  active.state = "cancelled";
  active.updated_at = new Date().toISOString();
  runtime.events.push(
    createExecutionEvent(conversation.id, active.id, active.queue_index, "execution_stopped", {
      reason: "user_stop"
    })
  );
  runtime.messages.push({
    id: createMockId("msg"),
    conversation_id: conversation.id,
    role: "system",
    content: "Execution stopped by user.",
    created_at: new Date().toISOString()
  });

  drainQueue(conversation.id, runtime);
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

  runtime.events.push(
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

  clearConversationTimer(conversationId);

  runtime.messages = snapshot.messages.map((message) => ({ ...message }));
  runtime.executions = runtime.executions
    .filter((execution) => snapshot.execution_ids.includes(execution.id))
    .filter((execution) => runtime.messages.some((message) => message.id === execution.message_id));
  runtime.snapshots = runtime.snapshots.filter((item) => item.created_at <= snapshot.created_at);
  runtime.worktreeRef = snapshot.worktree_ref;
  runtime.inspectorTab = snapshot.inspector_state.tab;
  runtime.diff = [];

  runtime.events.push(
    createExecutionEvent(conversationId, "", targetMessage.queue_index ?? 0, "thinking_delta", {
      stage: "snapshot_applied",
      message_id: messageId
    })
  );

  runtime.events.push(
    createExecutionEvent(conversationId, "", targetMessage.queue_index ?? 0, "thinking_delta", {
      stage: "rollback_completed",
      message_id: messageId
    })
  );

  drainQueue(conversationId, runtime);
}

export async function commitLatestDiff(conversationId: string): Promise<void> {
  const execution = getLatestFinishedExecution(conversationId);
  if (!execution) {
    return;
  }

  try {
    await commitExecution(execution.id);
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
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

function startOrQueueExecution(
  conversationId: string,
  runtime: ConversationRuntime,
  execution: Execution
): void {
  const active = runtime.executions.find((item) => item.state === "executing" || item.state === "confirming");
  if (active) {
    execution.state = "queued";
    execution.updated_at = new Date().toISOString();
    runtime.events.push(
      createExecutionEvent(conversationId, execution.id, execution.queue_index, "thinking_delta", {
        note: "queued_waiting"
      })
    );
    return;
  }

  runExecution(conversationId, runtime, execution);
}

function runExecution(
  conversationId: string,
  runtime: ConversationRuntime,
  execution: Execution
): void {
  execution.state = "executing";
  execution.updated_at = new Date().toISOString();
  runtime.events.push(
    createExecutionEvent(conversationId, execution.id, execution.queue_index, "execution_started", {
      mode: execution.mode
    })
  );

  conversationStore.timers[conversationId] = setTimeout(async () => {
    await completeExecution(conversationId, runtime, execution);
  }, 2200);
}

async function completeExecution(
  conversationId: string,
  runtime: ConversationRuntime,
  execution: Execution
): Promise<void> {
  execution.state = "completed";
  execution.updated_at = new Date().toISOString();

  runtime.messages.push({
    id: createMockId("msg"),
    conversation_id: conversationId,
    role: "assistant",
    content: `Execution ${execution.id} done. Queue index: ${execution.queue_index}.`,
    created_at: new Date().toISOString()
  });

  runtime.events.push(
    createExecutionEvent(conversationId, execution.id, execution.queue_index, "execution_done", {
      state: execution.state
    })
  );

  clearConversationTimer(conversationId);
  drainQueue(conversationId, runtime);

  try {
    runtime.diff = await loadExecutionDiff(execution.id);
    runtime.events.push(
      createExecutionEvent(conversationId, execution.id, execution.queue_index, "diff_generated", {
        files: runtime.diff.length
      })
    );
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  }
}

function drainQueue(conversationId: string, runtime: ConversationRuntime): void {
  const hasActive = runtime.executions.some((execution) => execution.state === "executing" || execution.state === "confirming");
  if (hasActive) {
    return;
  }

  const queued = runtime.executions
    .filter((execution) => execution.state === "queued")
    .sort((a, b) => a.queue_index - b.queue_index);
  if (queued.length === 0) {
    return;
  }

  runExecution(conversationId, runtime, queued[0]);
}
