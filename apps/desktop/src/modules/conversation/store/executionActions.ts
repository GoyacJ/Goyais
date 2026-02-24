import {
  cancelExecution,
  commitExecution,
  confirmExecution,
  createExecution,
  discardExecution,
  loadExecutionDiff,
  rollbackExecution
} from "@/modules/conversation/services";
import { createExecutionEvent } from "@/modules/conversation/store/events";
import {
  conversationStore,
  createConversationSnapshot,
  ensureConversationRuntime,
  findSnapshotForMessage,
  getLatestFinishedExecution,
  pushConversationSnapshot,
  type ConversationRuntime
} from "@/modules/conversation/store/state";
import { toDisplayError } from "@/shared/services/errorMapper";
import { createMockId } from "@/shared/services/mockData";
import type { Conversation, ConversationMessage, DiffItem, Execution, ExecutionEvent } from "@/shared/types/api";

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

    runtime.executions.push(response.execution);
    runtime.events.push(
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

  const active = runtime.executions.find((item) => item.state === "executing" || item.state === "confirming" || item.state === "pending");
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

export async function resolveExecutionConfirmation(executionId: string, decision: "approve" | "deny"): Promise<void> {
  try {
    await confirmExecution(executionId, decision);
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

  runtime.events.push(event);

  if (event.execution_id) {
    const execution = ensureExecution(runtime, conversationId, event);
    applyExecutionState(execution, event);
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

function ensureExecution(runtime: ConversationRuntime, conversationId: string, event: ExecutionEvent): Execution {
  let execution = runtime.executions.find((item) => item.id === event.execution_id);
  if (execution) {
    return execution;
  }

  execution = {
    id: event.execution_id,
    workspace_id: "",
    conversation_id: conversationId,
    message_id: "",
    state: "queued",
    mode: "agent",
    model_id: "",
    mode_snapshot: "agent",
    model_snapshot: {
      model_id: ""
    },
    project_revision_snapshot: 0,
    queue_index: event.queue_index,
    trace_id: event.trace_id,
    created_at: event.timestamp,
    updated_at: event.timestamp
  };
  runtime.executions.push(execution);
  return execution;
}

function applyExecutionState(execution: Execution, event: ExecutionEvent): void {
  switch (event.type) {
    case "execution_started":
      execution.state = "executing";
      break;
    case "confirmation_required":
      execution.state = "confirming";
      break;
    case "confirmation_resolved": {
      const decision = typeof event.payload.decision === "string" ? event.payload.decision.toLowerCase() : "";
      execution.state = decision === "deny" ? "cancelled" : "executing";
      break;
    }
    case "execution_stopped":
      execution.state = "cancelled";
      break;
    case "execution_done":
      execution.state = "completed";
      break;
    case "execution_error":
      execution.state = "failed";
      break;
    default:
      break;
  }
  execution.updated_at = event.timestamp;
}

function parseDiff(payload: Record<string, unknown>): DiffItem[] {
  const raw = payload.diff;
  if (!Array.isArray(raw)) {
    return [];
  }

  return raw
    .filter((item): item is Record<string, unknown> => typeof item === "object" && item !== null)
    .map((item) => ({
      id: typeof item.id === "string" ? item.id : createMockId("diff"),
      path: typeof item.path === "string" ? item.path : "unknown",
      change_type: item.change_type === "added" || item.change_type === "deleted" ? item.change_type : "modified",
      summary: typeof item.summary === "string" ? item.summary : "changed"
    }));
}
