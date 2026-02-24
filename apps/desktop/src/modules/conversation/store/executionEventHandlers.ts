import {
  applyExecutionState,
  dedupeExecutions,
  ensureExecution,
  parseDiff
} from "@/modules/conversation/store/executionRuntime";
import { shouldAppendTerminalMessage } from "@/modules/conversation/store/executionEventIdempotency";
import type { ConversationRuntime } from "@/modules/conversation/store/state";
import { createMockId } from "@/shared/services/mockData";
import type { ConversationMessage, ExecutionEvent } from "@/shared/types/api";

export type ExecutionTransition = {
  previousState: string | undefined;
  nextState: string | undefined;
  messageID: string;
};

export function updateExecutionTransition(
  runtime: ConversationRuntime,
  conversationId: string,
  event: ExecutionEvent
): ExecutionTransition {
  if (!event.execution_id) {
    return { previousState: undefined, nextState: undefined, messageID: "" };
  }

  const previousState = runtime.executions.find((item) => item.id === event.execution_id)?.state;
  const execution = ensureExecution(runtime, conversationId, event);
  applyExecutionState(execution, event);
  dedupeExecutions(runtime);
  const nextState = runtime.executions.find((item) => item.id === event.execution_id)?.state;
  return { previousState, nextState, messageID: execution.message_id };
}

export function applyDiffUpdate(runtime: ConversationRuntime, event: ExecutionEvent): void {
  if (event.type === "diff_generated") {
    runtime.diff = parseDiff(event.payload);
  }
}

export function appendTerminalMessageFromEvent(
  runtime: ConversationRuntime,
  conversationId: string,
  event: ExecutionEvent,
  transition: ExecutionTransition
): void {
  switch (event.type) {
    case "execution_done":
      appendExecutionDoneMessage(runtime, conversationId, event, transition);
      break;
    case "execution_error":
      appendExecutionErrorMessage(runtime, conversationId, event, transition);
      break;
    default:
      break;
  }
}

function appendExecutionDoneMessage(
  runtime: ConversationRuntime,
  conversationId: string,
  event: ExecutionEvent,
  transition: ExecutionTransition
): void {
  if (!shouldAppendTerminalMessage(runtime, event, transition.previousState, transition.nextState, transition.messageID, "assistant")) {
    return;
  }
  const content = asNonEmptyString(event.payload.content) || `Execution ${event.execution_id} completed.`;
  appendTerminalMessage(runtime, {
    id: createMockId("msg"),
    conversation_id: conversationId,
    role: "assistant",
    content,
    queue_index: event.queue_index,
    created_at: new Date().toISOString()
  });
}

function appendExecutionErrorMessage(
  runtime: ConversationRuntime,
  conversationId: string,
  event: ExecutionEvent,
  transition: ExecutionTransition
): void {
  if (!shouldAppendTerminalMessage(runtime, event, transition.previousState, transition.nextState, transition.messageID, "system")) {
    return;
  }
  const content = asNonEmptyString(event.payload.message) || "Execution failed.";
  appendTerminalMessage(runtime, {
    id: createMockId("msg"),
    conversation_id: conversationId,
    role: "system",
    content,
    queue_index: event.queue_index,
    created_at: new Date().toISOString()
  });
}

function asNonEmptyString(value: unknown): string {
  return typeof value === "string" && value.trim() !== "" ? value : "";
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
