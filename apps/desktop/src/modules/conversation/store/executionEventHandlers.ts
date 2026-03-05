import {
  applyRunState,
  dedupeExecutions,
  ensureExecution,
  parseDiff
} from "@/modules/conversation/store/executionRuntime";
import { shouldAppendTerminalMessage } from "@/modules/conversation/store/executionEventIdempotency";
import type { SessionRuntime } from "@/modules/conversation/store/state";
import { createMockId } from "@/shared/utils/id";
import type { RunLifecycleEvent, SessionMessage } from "@/shared/types/api";

export type ExecutionTransition = {
  previousState: string | undefined;
  nextState: string | undefined;
  messageID: string;
};

export function updateExecutionTransition(
  runtime: SessionRuntime,
  conversationId: string,
  event: RunLifecycleEvent
): ExecutionTransition {
  if (!event.execution_id) {
    return { previousState: undefined, nextState: undefined, messageID: "" };
  }

  const previousState = runtime.executions.find((item) => item.id === event.execution_id)?.state;
  const execution = ensureExecution(runtime, conversationId, event);
  applyRunState(execution, event);
  dedupeExecutions(runtime);
  const nextState = runtime.executions.find((item) => item.id === event.execution_id)?.state;
  return { previousState, nextState, messageID: execution.message_id };
}

export function applyDiffUpdate(runtime: SessionRuntime, event: RunLifecycleEvent): void {
  if (event.type === "diff_generated") {
    const incoming = parseDiff(event.payload);
    if (incoming.length > 0) {
      runtime.diff = mergeDiffByPath(runtime.diff, incoming);
    }
    return;
  }

  if (
    event.type === "change_set_committed" ||
    event.type === "change_set_discarded" ||
    event.type === "change_set_rolled_back"
  ) {
    runtime.diff = [];
  }
}

export function appendTerminalMessageFromEvent(
  runtime: SessionRuntime,
  conversationId: string,
  event: RunLifecycleEvent,
  transition: ExecutionTransition
): void {
  switch (event.type) {
    case "execution_done":
      appendExecutionDoneMessage(runtime, conversationId, event, transition);
      break;
    case "execution_error":
      appendExecutionErrorMessage(runtime, conversationId, event, transition);
      break;
    case "thinking_delta":
      appendUserAnswerMessage(runtime, conversationId, event);
      break;
    default:
      break;
  }
}

function appendExecutionDoneMessage(
  runtime: SessionRuntime,
  conversationId: string,
  event: RunLifecycleEvent,
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
  runtime: SessionRuntime,
  conversationId: string,
  event: RunLifecycleEvent,
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

function appendUserAnswerMessage(
  runtime: SessionRuntime,
  conversationId: string,
  event: RunLifecycleEvent
): void {
  const stage = asNonEmptyString(event.payload.stage);
  if (stage !== "run_user_question_resolved") {
    return;
  }
  const selectedOptionLabel = asNonEmptyString(event.payload.selected_option_label);
  const selectedOptionID = asNonEmptyString(event.payload.selected_option_id);
  const text = asNonEmptyString(event.payload.text);
  const question = asNonEmptyString(event.payload.question);
  const answerLine = selectedOptionLabel || selectedOptionID;
  const lines = [
    question ? `Question: ${question}` : "",
    answerLine ? `Answer: ${answerLine}` : "",
    text ? `Note: ${text}` : ""
  ].filter((item) => item !== "");
  if (lines.length === 0) {
    return;
  }
  const content = lines.join("\n");
  const duplicated = runtime.messages.some((message) =>
    message.role === "user" &&
    typeof message.queue_index === "number" &&
    message.queue_index === event.queue_index &&
    message.content.trim() === content
  );
  if (duplicated) {
    return;
  }
  appendTerminalMessage(runtime, {
    id: createMockId("msg"),
    conversation_id: conversationId,
    role: "user",
    content,
    queue_index: event.queue_index,
    created_at: new Date().toISOString()
  });
}

function asNonEmptyString(value: unknown): string {
  return typeof value === "string" && value.trim() !== "" ? value : "";
}

function appendTerminalMessage(runtime: SessionRuntime, message: SessionMessage): void {
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

function mergeDiffByPath(existing: SessionRuntime["diff"], incoming: SessionRuntime["diff"]): SessionRuntime["diff"] {
  const mergedByPath = new Map<string, SessionRuntime["diff"][number]>();
  for (const item of existing) {
    mergedByPath.set(item.path, item);
  }
  for (const item of incoming) {
    mergedByPath.set(item.path, item);
  }
  return [...mergedByPath.values()];
}
