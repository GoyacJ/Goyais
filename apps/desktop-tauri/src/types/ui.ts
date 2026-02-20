import type { EventEnvelope, EventType } from "@/types/generated";

export type CapabilityRisk = "none" | "write" | "exec" | "network" | "delete" | "exfil";

export interface RiskDetails {
  command?: string;
  cwd?: string;
  paths: string[];
  domains: string[];
  pathOutsideWorkspace: boolean;
}

export interface ToolCallView {
  callId: string;
  toolName: string;
  args: Record<string, unknown>;
  requiresConfirmation: boolean;
  status: "waiting" | "approved" | "denied" | "completed" | "failed";
  output?: unknown;
  createdAt: string;
  finishedAt?: string;
}

export type RunStreamState = "idle" | "streaming" | "waiting_confirmation" | "completed" | "failed";

export interface RunEventViewModel {
  id: string;
  seq: number;
  runId: string;
  ts: string;
  type: EventType;
  payload: Record<string, unknown>;
  payloadText: string;
  summary: string;
  streamState: RunStreamState;
  callId?: string;
  toolName?: string;
  raw: EventEnvelope;
}

export interface ToolDetailTabData {
  input: string;
  output: string;
  logs: string;
  timing: string;
}

export interface DiffHunkSelectionState {
  [hunkId: string]: boolean;
}
