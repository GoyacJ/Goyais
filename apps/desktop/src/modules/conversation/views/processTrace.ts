import type { Execution, ExecutionEvent, ExecutionState, TraceDetailLevel } from "@/shared/types/api";
import { isTerminalExecutionState } from "@/modules/conversation/store/executionMerge";

export type ExecutionTraceStep = {
  id: string;
  title: string;
  summary: string;
  details: string;
};

export type ExecutionTraceViewModel = {
  executionId: string;
  messageId: string;
  queueIndex: number;
  state: ExecutionState;
  isRunning: boolean;
  summary: string;
  steps: ExecutionTraceStep[];
};

const RENDERED_EVENT_TYPES: ReadonlySet<string> = new Set([
  "execution_started",
  "thinking_delta",
  "tool_call",
  "tool_result"
]);

export function buildExecutionTraceViewModels(
  events: ExecutionEvent[],
  executions: Execution[],
  now: Date = new Date()
): ExecutionTraceViewModel[] {
  const groupedEvents = groupExecutionEvents(events);
  return [...executions]
    .sort((left, right) => {
      if (left.queue_index !== right.queue_index) {
        return left.queue_index - right.queue_index;
      }
      return left.created_at.localeCompare(right.created_at);
    })
    .map((execution) => buildExecutionTraceViewModel(execution, groupedEvents.get(execution.id) ?? [], now));
}

function groupExecutionEvents(events: ExecutionEvent[]): Map<string, ExecutionEvent[]> {
  const grouped = new Map<string, ExecutionEvent[]>();
  for (const event of events) {
    if (!RENDERED_EVENT_TYPES.has(event.type)) {
      continue;
    }
    const executionId = event.execution_id.trim();
    if (executionId === "") {
      continue;
    }
    const list = grouped.get(executionId) ?? [];
    list.push(event);
    grouped.set(executionId, list);
  }
  for (const [executionId, list] of grouped.entries()) {
    grouped.set(executionId, sortEvents(list));
  }
  return grouped;
}

function buildExecutionTraceViewModel(
  execution: Execution,
  events: ExecutionEvent[],
  now: Date
): ExecutionTraceViewModel {
  const detailLevel = execution.agent_config_snapshot?.trace_detail_level ?? "verbose";
  const durationSec = resolveDurationSeconds(execution, events, now);
  const messageDurationSec = resolveMessageDurationSeconds(execution, now);
  const thinkingCount = events.filter((item) => item.type === "thinking_delta").length;
  const toolCallCount = events.filter((item) => item.type === "tool_call").length;
  const toolFailedCount = events.filter((item) => item.type === "tool_result" && item.payload.ok === false).length;
  const tokensIn = toNonNegativeInteger(execution.tokens_in);
  const tokensOut = toNonNegativeInteger(execution.tokens_out);
  const summary = buildSummary(
    execution.state,
    durationSec,
    messageDurationSec,
    thinkingCount,
    toolCallCount,
    toolFailedCount,
    tokensIn,
    tokensOut
  );

  return {
    executionId: execution.id,
    messageId: execution.message_id,
    queueIndex: execution.queue_index,
    state: execution.state,
    isRunning: execution.state === "pending" || execution.state === "executing",
    summary,
    steps: events.map((event, index) => toExecutionTraceStep(event, detailLevel, index))
  };
}

function sortEvents(events: ExecutionEvent[]): ExecutionEvent[] {
  return [...events].sort((left, right) => {
    if (left.sequence !== right.sequence) {
      return left.sequence - right.sequence;
    }
    return left.timestamp.localeCompare(right.timestamp);
  });
}

function resolveDurationSeconds(execution: Execution, events: ExecutionEvent[], now: Date): number {
  const startedAt = resolveStartedAt(execution, events);
  const endedAt = resolveEndedAt(execution, now);
  const durationMs = endedAt.getTime() - startedAt.getTime();
  if (!Number.isFinite(durationMs) || durationMs <= 0) {
    return 0;
  }
  return Math.round(durationMs / 1000);
}

function resolveMessageDurationSeconds(execution: Execution, now: Date): number {
  const startedAt = toDateOrNow(execution.created_at);
  const endedAt = resolveEndedAt(execution, now);
  const durationMs = endedAt.getTime() - startedAt.getTime();
  if (!Number.isFinite(durationMs) || durationMs <= 0) {
    return 0;
  }
  return Math.round(durationMs / 1000);
}

function resolveStartedAt(execution: Execution, events: ExecutionEvent[]): Date {
  const startedEvent = events.find((item) => item.type === "execution_started");
  const raw = startedEvent?.timestamp?.trim() || execution.created_at;
  return toDateOrNow(raw);
}

function resolveEndedAt(execution: Execution, now: Date): Date {
  if (!isTerminalExecutionState(execution.state)) {
    return now;
  }
  return toDateOrNow(execution.updated_at);
}

function toDateOrNow(input: string): Date {
  const value = new Date(input);
  if (Number.isNaN(value.getTime())) {
    return new Date();
  }
  return value;
}

function buildSummary(
  state: ExecutionState,
  durationSec: number,
  messageDurationSec: number,
  thinkingCount: number,
  toolCallCount: number,
  toolFailedCount: number,
  tokensIn: number | null,
  tokensOut: number | null
): string {
  if (state === "queued") {
    return "排队中，等待执行";
  }
  const tokenSummary = formatTokenSummary(tokensIn, tokensOut);
  const base = `已思考 ${durationSec}s，调用 ${toolCallCount} 个工具，Token ${tokenSummary}，消息执行 ${messageDurationSec}s`;
  if (state === "failed") {
    return toolFailedCount > 0 ? `执行失败，${base}，其中 ${toolFailedCount} 个失败` : `执行失败，${base}`;
  }
  if (state === "cancelled") {
    return `已停止，${base}`;
  }
  if (state === "pending" || state === "executing") {
    const thinkingText = thinkingCount > 0 ? `已思考 ${durationSec}s` : `执行中 ${durationSec}s`;
    return `${thinkingText}，已调用 ${toolCallCount} 个工具，Token ${tokenSummary}，消息执行 ${messageDurationSec}s`;
  }
  return base;
}

function formatTokenSummary(tokensIn: number | null, tokensOut: number | null): string {
  if (tokensIn === null || tokensOut === null) {
    return "N/A";
  }
  return `in ${tokensIn} / out ${tokensOut} / total ${tokensIn + tokensOut}`;
}

function toNonNegativeInteger(value: unknown): number | null {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return null;
  }
  if (value < 0) {
    return 0;
  }
  return Math.trunc(value);
}

function toExecutionTraceStep(event: ExecutionEvent, detailLevel: TraceDetailLevel, index: number): ExecutionTraceStep {
  const eventID = event.event_id?.trim() || `${event.execution_id}-${event.sequence}-${index}`;
  if (event.type === "execution_started") {
    return {
      id: eventID,
      title: "开始执行",
      summary: "execution_started",
      details: detailLevel === "verbose" ? toCompactJSON(event.payload) : ""
    };
  }
  if (event.type === "thinking_delta") {
    const stage = asString(event.payload.stage) || "thinking";
    const delta = asString(event.payload.delta);
    return {
      id: eventID,
      title: "思考",
      summary: detailLevel === "verbose" && delta ? `${stage}: ${truncate(delta, 160)}` : stage,
      details: detailLevel === "verbose" ? toCompactJSON(event.payload) : ""
    };
  }
  if (event.type === "tool_call") {
    const name = asString(event.payload.name) || "tool";
    const riskLevel = asString(event.payload.risk_level);
    return {
      id: eventID,
      title: "工具调用",
      summary: riskLevel ? `${name} (${riskLevel})` : name,
      details: detailLevel === "verbose" ? toCompactJSON(event.payload) : ""
    };
  }
  const toolName = asString(event.payload.name) || "tool";
  const okText = event.payload.ok === false ? "failed" : "done";
  return {
    id: eventID,
    title: "工具结果",
    summary: `${toolName} (${okText})`,
    details: detailLevel === "verbose" ? toCompactJSON(event.payload) : ""
  };
}

function asString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function truncate(value: string, maxLength: number): string {
  if (value.length <= maxLength) {
    return value;
  }
  return `${value.slice(0, maxLength)}...`;
}

function toCompactJSON(value: unknown): string {
  try {
    return truncate(JSON.stringify(value, null, 2), 1200);
  } catch {
    return "";
  }
}
