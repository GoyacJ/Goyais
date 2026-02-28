import type { Locale } from "@/shared/i18n/messages";
import type { Execution } from "@/shared/types/api";

export type TraceLocale = Locale;

export type TraceEventType = "execution_started" | "thinking_delta" | "tool_call" | "tool_result";

export type NormalizedThinkingStage =
  | "model_call"
  | "assistant_output"
  | "run_approval_needed"
  | "approval_granted"
  | "approval_denied"
  | "approval_resolved"
  | "turn_limit_reached"
  | "other";

export type NormalizedTraceEvent = {
  id: string;
  executionId: string;
  queueIndex: number;
  sequence: number;
  timestamp: string;
  type: TraceEventType;
  stage: NormalizedThinkingStage;
  payload: Record<string, unknown>;
  rawPayload: string;
  reasoningSentence: string;
  operationSummary: string;
  resultSummary: string;
  riskLevel: string;
  toolName: string;
  callId: string;
  isSuccess: boolean | null;
};

export type NormalizedExecutionTrace = {
  execution: Execution;
  events: NormalizedTraceEvent[];
};

export type TraceStepKind = "lifecycle" | "reasoning" | "tool_call" | "tool_result";

export type TraceStatusTone = "neutral" | "success" | "warning" | "error";

export type ExecutionTraceStepViewModel = {
  id: string;
  kind: TraceStepKind;
  title: string;
  summary: string;
  detail: string;
  timestampLabel: string;
  statusTone: TraceStatusTone;
  rawPayload: string;
};

export type ExecutionTraceViewModelData = {
  executionId: string;
  messageId: string;
  queueIndex: number;
  state: Execution["state"];
  isRunning: boolean;
  summaryPrimary: string;
  summarySecondary: string;
  steps: ExecutionTraceStepViewModel[];
};

export type RunningActionType = "model" | "tool" | "subagent" | "approval";

export type RunningActionBaseViewModel = {
  actionId: string;
  executionId: string;
  queueIndex: number;
  type: RunningActionType;
  primary: string;
  secondary: string;
  startedAt: string;
};

export type RunningActionViewModelData = RunningActionBaseViewModel & {
  elapsedMs: number;
  elapsedLabel: string;
};
