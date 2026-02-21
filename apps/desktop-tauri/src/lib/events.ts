import type { EventEnvelope } from "@/types/generated";
import type { RunEventViewModel, RunStreamState } from "@/types/ui";

function toText(payload: Record<string, unknown>): string {
  return JSON.stringify(payload, null, 2);
}

function streamStateFrom(event: EventEnvelope): RunStreamState {
  if (event.type === "error") return "failed";
  if (event.type === "done") {
    return event.payload.status === "completed" ? "completed" : "failed";
  }
  if (event.type === "tool_call" && event.payload.requires_confirmation === true) {
    return "waiting_confirmation";
  }
  return "streaming";
}

function summaryFrom(event: EventEnvelope): string {
  const payload = event.payload as Record<string, unknown>;
  if (event.type === "tool_call") {
    return `Tool call: ${String(payload.tool_name ?? "unknown")}`;
  }
  if (event.type === "tool_result") {
    const ok = payload.ok === true ? "ok" : "error";
    return `Tool result: ${String(payload.call_id ?? "unknown")} (${ok})`;
  }
  if (event.type === "patch") {
    return "Patch generated";
  }
  if (event.type === "error") {
    const error = payload.error as Record<string, unknown> | undefined;
    return `Error: ${String(error?.message ?? "unknown")}`;
  }
  if (event.type === "done") {
    return `Execution ${String(payload.status ?? "finished")}`;
  }
  if (event.type === "plan") {
    return String(payload.summary ?? "Plan updated");
  }
  return event.type;
}

export function normalizeEventEnvelope(event: EventEnvelope): RunEventViewModel {
  const payload = event.payload as Record<string, unknown>;
  return {
    id: event.event_id,
    seq: event.seq,
    executionId: event.execution_id,
    ts: event.ts,
    type: event.type,
    payload,
    payloadText: toText(payload),
    summary: summaryFrom(event),
    streamState: streamStateFrom(event),
    callId: typeof payload.call_id === "string" ? payload.call_id : undefined,
    toolName: typeof payload.tool_name === "string" ? payload.tool_name : undefined,
    raw: event
  };
}
