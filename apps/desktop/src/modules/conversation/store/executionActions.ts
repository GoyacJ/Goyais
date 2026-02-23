import {
  cancelExecution,
  commitExecution,
  createExecution,
  discardExecution,
  loadExecutionDiff
} from "@/modules/conversation/services";
import { createExecutionEvent } from "@/modules/conversation/store/events";
import {
  clearConversationTimer,
  conversationStore,
  countActiveAndQueued,
  ensureConversationRuntime,
  getLatestFinishedExecution,
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

  try {
    const response = await createExecution(conversation, {
      content,
      mode: runtime.mode,
      model_id: runtime.modelId
    });

    const execution = {
      ...response.execution,
      queue_index: queueIndex
    };

    runtime.executions.push(execution);
    runtime.events.push(
      createExecutionEvent(conversation.id, execution.id, queueIndex, "message_received", {
        message_id: userMessage.id
      })
    );
    startOrQueueExecution(conversation, runtime, execution);
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

  drainQueue(conversation, runtime);
}

export function rollbackConversationToMessage(conversationId: string, messageId: string): void {
  const runtime = conversationStore.byConversationId[conversationId];
  if (!runtime) {
    return;
  }

  const targetIndex = runtime.messages.findIndex((message) => message.id === messageId);
  if (targetIndex < 0) {
    return;
  }

  const targetMessage = runtime.messages[targetIndex];
  if (targetMessage.role !== "user") {
    return;
  }

  runtime.messages = runtime.messages.slice(0, targetIndex + 1);
  runtime.executions = runtime.executions.filter((execution) =>
    runtime.messages.some((message) => message.id === execution.message_id)
  );
  clearConversationTimer(conversationId);
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
  conversation: Conversation,
  runtime: ConversationRuntime,
  execution: Execution
): void {
  const active = runtime.executions.find((item) => item.state === "executing" || item.state === "confirming");
  if (active) {
    execution.state = "queued";
    execution.updated_at = new Date().toISOString();
    runtime.events.push(
      createExecutionEvent(conversation.id, execution.id, execution.queue_index, "thinking_delta", {
        note: "queued_waiting"
      })
    );
    return;
  }

  runExecution(conversation, runtime, execution);
}

function runExecution(
  conversation: Conversation,
  runtime: ConversationRuntime,
  execution: Execution
): void {
  execution.state = "executing";
  execution.updated_at = new Date().toISOString();
  runtime.events.push(
    createExecutionEvent(conversation.id, execution.id, execution.queue_index, "execution_started", {
      mode: execution.mode
    })
  );

  conversationStore.timers[conversation.id] = setTimeout(async () => {
    await completeExecution(conversation, runtime, execution);
  }, 2200);
}

async function completeExecution(
  conversation: Conversation,
  runtime: ConversationRuntime,
  execution: Execution
): Promise<void> {
  execution.state = "completed";
  execution.updated_at = new Date().toISOString();

  runtime.messages.push({
    id: createMockId("msg"),
    conversation_id: conversation.id,
    role: "assistant",
    content: `Execution ${execution.id} done. Queue index: ${execution.queue_index}.`,
    created_at: new Date().toISOString()
  });

  runtime.events.push(
    createExecutionEvent(conversation.id, execution.id, execution.queue_index, "execution_done", {
      state: execution.state
    })
  );

  try {
    runtime.diff = await loadExecutionDiff(execution.id);
    runtime.events.push(
      createExecutionEvent(conversation.id, execution.id, execution.queue_index, "diff_generated", {
        files: runtime.diff.length
      })
    );
  } catch (error) {
    conversationStore.error = toDisplayError(error);
  } finally {
    clearConversationTimer(conversation.id);
    drainQueue(conversation, runtime);
  }
}

function drainQueue(conversation: Conversation, runtime: ConversationRuntime): void {
  const queued = runtime.executions
    .filter((execution) => execution.state === "queued")
    .sort((a, b) => a.queue_index - b.queue_index);
  if (queued.length === 0) {
    return;
  }

  runExecution(conversation, runtime, queued[0]);
}
