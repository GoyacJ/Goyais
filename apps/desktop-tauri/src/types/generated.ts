export type EventType = "plan" | "tool_call" | "tool_result" | "patch" | "error" | "done";

export interface GoyaisError {
  code: string;
  message: string;
  trace_id: string;
  retryable: boolean;
  details?: Record<string, unknown>;
  cause?: string;
  ts?: string;
}

export interface EventEnvelope {
  protocol_version: "2.0.0";
  trace_id: string;
  event_id: string;
  run_id: string;
  seq: number;
  ts: string;
  type: EventType;
  payload: Record<string, unknown>;
}
