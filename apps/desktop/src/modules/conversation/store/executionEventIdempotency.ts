import type { ConversationRuntime } from "@/modules/conversation/store/state";
import type { ExecutionEvent } from "@/shared/types/api";

const MAX_PROCESSED_EVENT_KEYS = 5000;
const MAX_COMPLETION_MESSAGE_KEYS = 2000;

export function buildEventDedupKey(event: ExecutionEvent): string {
  const eventID = event.event_id?.trim();
  if (eventID !== "") {
    return `id:${eventID}`;
  }
  return `fallback:${event.conversation_id}:${event.execution_id}:${event.sequence}:${event.type}`;
}

export function rememberProcessedEvent(runtime: ConversationRuntime, eventDedupKey: string): void {
  if (eventDedupKey === "") {
    return;
  }
  runtime.processedEventKeySet.add(eventDedupKey);
  runtime.processedEventKeys.push(eventDedupKey);
  if (runtime.processedEventKeys.length > MAX_PROCESSED_EVENT_KEYS) {
    const overflow = runtime.processedEventKeys.length - MAX_PROCESSED_EVENT_KEYS;
    const stale = runtime.processedEventKeys.splice(0, overflow);
    for (const key of stale) {
      runtime.processedEventKeySet.delete(key);
    }
  }
}

export function shouldAppendTerminalMessage(
  runtime: ConversationRuntime,
  event: ExecutionEvent,
  previousState: string | undefined,
  nextState: string | undefined,
  messageID: string,
  role: "assistant" | "system"
): boolean {
  if (!isTerminalState(nextState)) {
    return false;
  }
  if (isTerminalState(previousState)) {
    return false;
  }
  if (!hasMessageContext(runtime, event, messageID)) {
    return false;
  }

  const completionKey = buildCompletionKey(event, role);
  if (runtime.completionMessageKeySet.has(completionKey)) {
    return false;
  }
  rememberCompletionMessage(runtime, completionKey);
  return true;
}

function rememberCompletionMessage(runtime: ConversationRuntime, completionKey: string): void {
  runtime.completionMessageKeySet.add(completionKey);
  runtime.completionMessageKeys.push(completionKey);
  if (runtime.completionMessageKeys.length > MAX_COMPLETION_MESSAGE_KEYS) {
    const overflow = runtime.completionMessageKeys.length - MAX_COMPLETION_MESSAGE_KEYS;
    const stale = runtime.completionMessageKeys.splice(0, overflow);
    for (const key of stale) {
      runtime.completionMessageKeySet.delete(key);
    }
  }
}

function hasMessageContext(runtime: ConversationRuntime, event: ExecutionEvent, messageID: string): boolean {
  const normalizedMessageID = messageID.trim();
  if (normalizedMessageID !== "" && runtime.messages.some((item) => item.id === normalizedMessageID)) {
    return true;
  }
  return runtime.messages.some(
    (item) => item.role === "user" && typeof item.queue_index === "number" && item.queue_index === event.queue_index
  );
}

function buildCompletionKey(event: ExecutionEvent, role: "assistant" | "system"): string {
  const eventID = event.event_id?.trim();
  if (eventID !== "") {
    return `${role}:id:${eventID}`;
  }
  return `${role}:fallback:${event.execution_id}:${event.type}:${event.sequence}`;
}

function isTerminalState(state: string | undefined): boolean {
  return state === "completed" || state === "failed" || state === "cancelled";
}
