export interface SyncEventEnvelope {
  protocol_version: "1.0.0";
  event_id: string;
  run_id: string;
  seq: number;
  ts: string;
  type: "plan" | "tool_call" | "tool_result" | "patch" | "error" | "done";
  payload: Record<string, unknown>;
}

export interface PushRequest {
  device_id: string;
  since_global_seq: number;
  events: Array<SyncEventEnvelope & { global_seq?: number }>;
  artifacts_meta: Array<Record<string, unknown>>;
}
