import type { Locale } from "@/shared/i18n/messages";
import type { Run } from "@/shared/types/api";

export type TraceLocale = Locale;

export type TraceEventType = "execution_started" | "thinking_delta" | "tool_call" | "tool_result";

export type NormalizedThinkingStage =
  | "model_call"
  | "assistant_output"
  | "run_approval_needed"
  | "run_user_question_needed"
  | "run_user_question_resolved"
  | "approval_granted"
  | "approval_denied"
  | "approval_resolved"
  | "turn_limit_reached"
  | "other";

export type OperationIntentKind = "command" | "path" | "url" | "query" | "scalar" | "none";

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
  operationIntentKind: OperationIntentKind;
  operationIntentValue: string;
  resultSummary: string;
  riskLevel: string;
  toolName: string;
  callId: string;
  isSuccess: boolean | null;
};

export type NormalizedRunTrace = {
  execution: Run;
  events: NormalizedTraceEvent[];
};

export type TraceStepKind = "lifecycle" | "reasoning" | "tool_call" | "tool_result";

export type TraceStatusTone = "neutral" | "success" | "warning" | "error";
export type TraceSummaryTone = "primary" | "success" | "warning" | "error" | "neutral";

export type RunTraceStepViewModel = {
  id: string;
  kind: TraceStepKind;
  title: string;
  summary: string;
  detail: string;
  timestampLabel: string;
  statusTone: TraceStatusTone;
  rawPayload: string;
};

export type RunTraceViewModelData = {
  executionId: string;
  messageId: string;
  queueIndex: number;
  state: Run["state"];
  isRunning: boolean;
  summaryPrimary: string;
  summarySecondary: string;
  summaryTone: TraceSummaryTone;
  steps: RunTraceStepViewModel[];
};

export type RunningActionType = "model" | "tool" | "subagent" | "approval" | "user_input";

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
