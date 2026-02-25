import type { ExecutionEvent, ExecutionEventType, RunEventType } from "@/shared/types/api";

const executionEventTypes: readonly ExecutionEventType[] = [
  "message_received",
  "execution_started",
  "thinking_delta",
  "tool_call",
  "tool_result",
  "diff_generated",
  "execution_stopped",
  "execution_done",
  "execution_error"
];

const runEventTypes: readonly RunEventType[] = [
  "run_queued",
  "run_started",
  "run_output_delta",
  "run_approval_needed",
  "run_completed",
  "run_failed",
  "run_cancelled"
];

const runToExecutionTypeMap: Record<Exclude<RunEventType, "run_output_delta" | "run_approval_needed">, ExecutionEventType> = {
  run_queued: "message_received",
  run_started: "execution_started",
  run_completed: "execution_done",
  run_failed: "execution_error",
  run_cancelled: "execution_stopped"
};

export function toExecutionEventFromStreamPayload(raw: unknown, fallbackConversationId: string): ExecutionEvent | null {
  if (!isRecord(raw)) {
    return null;
  }

  const rawType = asString(raw.type);
  if (rawType === "") {
    return null;
  }

  if (isRunEventType(rawType)) {
    return mapRunEventToExecutionEvent(raw, rawType, fallbackConversationId);
  }

  if (!isExecutionEventType(rawType)) {
    return null;
  }
  return normalizeLegacyExecutionEvent(raw, rawType, fallbackConversationId);
}

function normalizeLegacyExecutionEvent(
  raw: Record<string, unknown>,
  eventType: ExecutionEventType,
  fallbackConversationId: string
): ExecutionEvent {
  const payload = asRecord(raw.payload);
  return {
    event_id: asString(raw.event_id),
    execution_id: asString(raw.execution_id),
    conversation_id: resolveConversationId(asString(raw.conversation_id), fallbackConversationId),
    trace_id: asString(raw.trace_id),
    sequence: asInteger(raw.sequence, 0),
    queue_index: asInteger(raw.queue_index, asInteger(payload.queue_index, 0)),
    type: eventType,
    timestamp: asTimestamp(raw.timestamp),
    payload
  };
}

function mapRunEventToExecutionEvent(
  raw: Record<string, unknown>,
  runType: RunEventType,
  fallbackConversationId: string
): ExecutionEvent {
  const payload = asRecord(raw.payload);
  const queueIndex = asInteger(raw.queue_index, asInteger(payload.queue_index, 0));
  const traceId = asString(raw.trace_id) || asString(payload.trace_id);
  const conversationId = resolveConversationId(asString(raw.session_id), fallbackConversationId);

  if (runType === "run_output_delta") {
    return {
      event_id: asString(raw.event_id),
      execution_id: asString(raw.run_id),
      conversation_id: conversationId,
      trace_id: traceId,
      sequence: asInteger(raw.sequence, 0),
      queue_index: queueIndex,
      type: resolveExecutionTypeForRunOutputDelta(payload),
      timestamp: asTimestamp(raw.timestamp),
      payload
    };
  }

  if (runType === "run_approval_needed") {
    return {
      event_id: asString(raw.event_id),
      execution_id: asString(raw.run_id),
      conversation_id: conversationId,
      trace_id: traceId,
      sequence: asInteger(raw.sequence, 0),
      queue_index: queueIndex,
      type: "thinking_delta",
      timestamp: asTimestamp(raw.timestamp),
      payload: {
        ...payload,
        stage: asString(payload.stage) || "run_approval_needed",
        run_state: asString(payload.run_state) || "waiting_approval"
      }
    };
  }

  return {
    event_id: asString(raw.event_id),
    execution_id: asString(raw.run_id),
    conversation_id: conversationId,
    trace_id: traceId,
    sequence: asInteger(raw.sequence, 0),
    queue_index: queueIndex,
    type: runToExecutionTypeMap[runType],
    timestamp: asTimestamp(raw.timestamp),
    payload
  };
}

function resolveExecutionTypeForRunOutputDelta(payload: Record<string, unknown>): ExecutionEventType {
  if (Array.isArray(payload.diff)) {
    return "diff_generated";
  }
  if (asString(payload.call_id) !== "") {
    if (payload.output !== undefined || typeof payload.ok === "boolean") {
      return "tool_result";
    }
    return "tool_call";
  }

  const hasToolName = asString(payload.name) !== "";
  if (hasToolName && payload.output !== undefined) {
    return "tool_result";
  }
  if (hasToolName && payload.input !== undefined) {
    return "tool_call";
  }
  return "thinking_delta";
}

function resolveConversationId(rawConversationId: string, fallbackConversationId: string): string {
  const trimmed = rawConversationId.trim();
  if (trimmed !== "") {
    return trimmed;
  }
  return fallbackConversationId.trim();
}

function isExecutionEventType(value: string): value is ExecutionEventType {
  return executionEventTypes.includes(value as ExecutionEventType);
}

function isRunEventType(value: string): value is RunEventType {
  return runEventTypes.includes(value as RunEventType);
}

function asString(value: unknown): string {
  if (typeof value !== "string") {
    return "";
  }
  return value.trim();
}

function asInteger(value: unknown, fallback: number): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return Math.trunc(value);
  }
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number.parseInt(value, 10);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  return fallback;
}

function asTimestamp(value: unknown): string {
  const trimmed = asString(value);
  if (trimmed !== "") {
    return trimmed;
  }
  return new Date().toISOString();
}

function asRecord(value: unknown): Record<string, unknown> {
  if (!isRecord(value)) {
    return {};
  }
  return value;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
