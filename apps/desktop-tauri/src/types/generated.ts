export type EventType = "plan" | "tool_call" | "tool_result" | "patch" | "error" | "done";

export interface EventEnvelope {
  protocol_version: "1.0.0";
  event_id: string;
  run_id: string;
  seq: number;
  ts: string;
  type: EventType;
  payload: Record<string, unknown>;
}
