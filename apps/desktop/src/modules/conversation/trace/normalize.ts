import type { ExecutionEvent } from "@/shared/types/api";

import type { NormalizedThinkingStage, NormalizedTraceEvent, TraceEventType } from "@/modules/conversation/trace/types";
import {
  asString,
  extractOperationSummary,
  extractReasoningSentence,
  extractResultSummary,
  redactSensitivePayload,
  toCompactJSON
} from "@/modules/conversation/trace/summarize";

const RENDERED_EVENT_TYPES: ReadonlySet<string> = new Set([
  "execution_started",
  "thinking_delta",
  "tool_call",
  "tool_result"
]);

export function normalizeExecutionEventsByExecution(events: ExecutionEvent[]): Map<string, NormalizedTraceEvent[]> {
  const grouped = new Map<string, NormalizedTraceEvent[]>();

  for (let index = 0; index < events.length; index += 1) {
    const event = events[index];
    if (!event || !RENDERED_EVENT_TYPES.has(event.type)) {
      continue;
    }

    const executionId = event.execution_id.trim();
    if (executionId === "") {
      continue;
    }

    const normalized = normalizeExecutionEvent(event, index);
    const list = grouped.get(executionId) ?? [];
    list.push(normalized);
    grouped.set(executionId, list);
  }

  for (const [executionId, list] of grouped.entries()) {
    grouped.set(executionId, sortEvents(list));
  }

  return grouped;
}

function normalizeExecutionEvent(event: ExecutionEvent, index: number): NormalizedTraceEvent {
  const type = event.type as TraceEventType;
  const payload = redactSensitivePayload(event.payload ?? {});
  const stage = type === "thinking_delta"
    ? normalizeThinkingStage(asString(payload.stage))
    : "other";
  const callId = asString(payload.call_id);
  const toolName = asString(payload.name);

  const isSuccess = type === "tool_result"
    ? payload.ok === false
      ? false
      : payload.ok === true
        ? true
        : null
    : null;

  const operationSummary = type === "tool_call" || stage === "run_approval_needed"
    ? extractOperationSummary(payload)
    : "";

  const resultSummary = type === "tool_result"
    ? extractResultSummary(payload, isSuccess)
    : "";

  const reasoningSentence = type === "thinking_delta"
    ? extractReasoningSentence(asString(payload.delta), asString(payload.stage) || "thinking")
    : "";

  const eventId = event.event_id.trim() || `${event.execution_id}-${event.sequence}-${index}`;

  return {
    id: eventId,
    executionId: event.execution_id,
    queueIndex: event.queue_index,
    sequence: event.sequence,
    timestamp: event.timestamp,
    type,
    stage,
    payload,
    rawPayload: toCompactJSON(payload, 1500),
    reasoningSentence,
    operationSummary,
    resultSummary,
    riskLevel: asString(payload.risk_level).toLowerCase(),
    toolName,
    callId,
    isSuccess
  };
}

function normalizeThinkingStage(value: string): NormalizedThinkingStage {
  switch (value) {
    case "model_call":
    case "assistant_output":
    case "run_approval_needed":
    case "approval_granted":
    case "approval_denied":
    case "approval_resolved":
    case "turn_limit_reached":
      return value;
    default:
      return "other";
  }
}

function sortEvents(events: NormalizedTraceEvent[]): NormalizedTraceEvent[] {
  return [...events].sort((left, right) => {
    if (left.sequence !== right.sequence) {
      return left.sequence - right.sequence;
    }
    return left.timestamp.localeCompare(right.timestamp);
  });
}
