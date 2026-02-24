import type { ExecutionEvent, TraceDetailLevel } from "@/shared/types/api";

export type ProcessTraceItem = {
  id: string;
  title: string;
  summary: string;
  details: string;
};

export function buildProcessTraceItems(
  events: ExecutionEvent[],
  executionId: string,
  detailLevel: TraceDetailLevel
): ProcessTraceItem[] {
  return events
    .filter((event) => event.execution_id === executionId)
    .filter((event) => event.type === "execution_started" || event.type === "thinking_delta" || event.type === "tool_call" || event.type === "tool_result")
    .sort((left, right) => {
      if (left.sequence !== right.sequence) {
        return left.sequence - right.sequence;
      }
      return left.timestamp.localeCompare(right.timestamp);
    })
    .map((event, index) => toProcessTraceItem(event, detailLevel, index));
}

function toProcessTraceItem(event: ExecutionEvent, detailLevel: TraceDetailLevel, index: number): ProcessTraceItem {
  const eventID = event.event_id?.trim() || `${event.execution_id}-${event.sequence}-${index}`;
  if (event.type === "execution_started") {
    return {
      id: eventID,
      title: "Execution Started",
      summary: "开始执行",
      details: detailLevel === "verbose" ? toCompactJSON(event.payload) : ""
    };
  }
  if (event.type === "thinking_delta") {
    const stage = asString(event.payload.stage) || "thinking";
    const delta = asString(event.payload.delta);
    return {
      id: eventID,
      title: "Thinking",
      summary: detailLevel === "verbose" && delta ? `${stage}: ${truncate(delta, 160)}` : stage,
      details: detailLevel === "verbose" ? toCompactJSON(event.payload) : ""
    };
  }
  if (event.type === "tool_call") {
    const name = asString(event.payload.name) || "tool";
    const riskLevel = asString(event.payload.risk_level);
    return {
      id: eventID,
      title: "Tool Call",
      summary: riskLevel ? `${name} (${riskLevel})` : name,
      details: detailLevel === "verbose" ? toCompactJSON(event.payload) : ""
    };
  }
  const toolName = asString(event.payload.name) || "tool";
  const okText = event.payload.ok === false ? "failed" : "done";
  return {
    id: eventID,
    title: "Tool Result",
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
