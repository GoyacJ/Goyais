import { isTerminalRunState } from "@/modules/conversation/store/executionMerge";
import { normalizeExecutionEventsByExecution } from "@/modules/conversation/trace/normalize";
import { truncateText } from "@/modules/conversation/trace/summarize";
import type {
  ExecutionTraceStepViewModel,
  ExecutionTraceViewModelData,
  NormalizedThinkingStage,
  NormalizedTraceEvent,
  OperationIntentKind,
  RunningActionBaseViewModel,
  RunningActionType,
  RunningActionViewModelData,
  TraceLocale,
  TraceSummaryTone,
  TraceStatusTone
} from "@/modules/conversation/trace/types";
import { messages } from "@/shared/i18n/messages";
import type { Run, RunLifecycleEvent, TraceDetailLevel } from "@/shared/types/api";

type ActiveAction = RunningActionBaseViewModel & {
  toolName: string;
  comparisonName: string;
};

export function buildExecutionTraceViewModelData(
  events: RunLifecycleEvent[],
  executions: Run[],
  locale: TraceLocale,
  now: Date = new Date()
): ExecutionTraceViewModelData[] {
  const groupedEvents = normalizeExecutionEventsByExecution(events);

  return [...executions]
    .sort((left, right) => {
      if (left.queue_index !== right.queue_index) {
        return left.queue_index - right.queue_index;
      }
      return left.created_at.localeCompare(right.created_at);
    })
    .map((execution) => {
      const normalizedEvents = groupedEvents.get(execution.id) ?? [];
      const summary = buildTraceSummary(execution, normalizedEvents, locale, now);
      const detailLevel = execution.agent_config_snapshot?.trace_detail_level ?? "verbose";
      const visibleStepEvents = normalizedEvents.filter((event) => isMeaningfulTraceStepEvent(event));

      return {
        executionId: execution.id,
        messageId: execution.message_id,
        queueIndex: execution.queue_index,
        state: execution.state,
        isRunning:
          execution.state === "pending" ||
          execution.state === "executing" ||
          execution.state === "confirming" ||
          execution.state === "awaiting_input",
        summaryPrimary: summary.primary,
        summarySecondary: summary.secondary,
        summaryTone: summary.tone,
        steps: visibleStepEvents.map((event, index) => toTraceStep(event, detailLevel, locale, index))
      };
    });
}

export function buildRunningActionBaseViewModelData(
  events: RunLifecycleEvent[],
  executions: Run[],
  locale: TraceLocale
): RunningActionBaseViewModel[] {
  const groupedEvents = normalizeExecutionEventsByExecution(events);
  const runningExecutions = executions
    .filter(
      (execution) =>
        execution.state === "pending" ||
        execution.state === "executing" ||
        execution.state === "confirming" ||
        execution.state === "awaiting_input"
    )
    .sort((left, right) => {
      if (left.queue_index !== right.queue_index) {
        return left.queue_index - right.queue_index;
      }
      return left.created_at.localeCompare(right.created_at);
    });

  const actions = runningExecutions.flatMap((execution) =>
    collectActiveActionsForExecution(execution, groupedEvents.get(execution.id) ?? [], locale)
  );

  return actions.sort((left, right) => {
    if (left.queueIndex !== right.queueIndex) {
      return left.queueIndex - right.queueIndex;
    }
    return left.startedAt.localeCompare(right.startedAt);
  });
}

export function hydrateRunningActionElapsed(
  actions: RunningActionBaseViewModel[],
  locale: TraceLocale,
  now: Date = new Date()
): RunningActionViewModelData[] {
  return actions.map((action) => {
    const elapsedMs = Math.max(0, now.getTime() - toDateOrNow(action.startedAt).getTime());
    const seconds = Math.floor(elapsedMs / 1000);
    return {
      ...action,
      elapsedMs,
      elapsedLabel: tr(locale, "conversation.running.elapsed", { seconds })
    };
  });
}

function collectActiveActionsForExecution(
  execution: Run,
  events: NormalizedTraceEvent[],
  locale: TraceLocale
): RunningActionBaseViewModel[] {
  const activeActions = new Map<string, ActiveAction>();
  const activeModelActionIds: string[] = [];
  let latestReasoningSentence = "";

  for (const event of events) {
    if (event.reasoningSentence !== "" && event.stage !== "model_call") {
      latestReasoningSentence = event.reasoningSentence;
    }

    if (event.type === "thinking_delta") {
      handleThinkingDeltaEvent(execution, event, locale, activeActions, activeModelActionIds, latestReasoningSentence);
      continue;
    }
    if (event.type === "tool_call") {
      handleToolCallEvent(execution, event, locale, activeActions, latestReasoningSentence);
      continue;
    }
    if (event.type === "tool_result") {
      handleToolResultEvent(execution, event, activeActions);
    }
  }

  return [...activeActions.values()];
}

function handleThinkingDeltaEvent(
  execution: Run,
  event: NormalizedTraceEvent,
  locale: TraceLocale,
  activeActions: Map<string, ActiveAction>,
  activeModelActionIds: string[],
  latestReasoningSentence: string
): void {
  if (event.stage === "run_approval_needed") {
    const actionId = event.callId !== ""
      ? `approval:${execution.id}:${event.callId}`
      : `approval:${execution.id}:seq:${event.sequence}`;
    const toolName = resolveToolName(locale, event.toolName);
    activeActions.set(actionId, {
      actionId,
      executionId: execution.id,
      queueIndex: execution.queue_index,
      type: "approval",
      toolName,
      comparisonName: "",
      primary: tr(locale, "conversation.running.primary.approval", { tool: toolName }),
      secondary: composeSecondary(locale, latestReasoningSentence, event.operationSummary),
      startedAt: event.timestamp
    });
    return;
  }

  if (event.stage === "run_user_question_needed") {
    const questionID = String(event.payload.question_id ?? "").trim();
    const actionId = questionID !== ""
      ? `user_input:${execution.id}:${questionID}`
      : `user_input:${execution.id}:seq:${event.sequence}`;
    const question = String(event.payload.question ?? "").trim();
    activeActions.set(actionId, {
      actionId,
      executionId: execution.id,
      queueIndex: execution.queue_index,
      type: "user_input",
      toolName: "",
      comparisonName: "",
      primary: question !== ""
        ? tr(locale, "conversation.running.primary.userInputQuestion", { question: truncateText(question, 72) })
        : tr(locale, "conversation.running.primary.userInput"),
      secondary: composeSecondary(locale, latestReasoningSentence, event.operationSummary),
      startedAt: event.timestamp
    });
    return;
  }

  if (
    event.stage === "approval_granted" ||
    event.stage === "approval_denied" ||
    event.stage === "approval_resolved" ||
    event.stage === "run_user_question_resolved"
  ) {
    for (const action of [...activeActions.values()]) {
      if (action.type === "approval" || action.type === "user_input") {
        activeActions.delete(action.actionId);
      }
    }
    return;
  }

  if (event.stage === "model_call") {
    const actionId = `model:${execution.id}:${event.sequence}`;
    const modelReasoning = truncateText(
      event.reasoningSentence !== "" ? event.reasoningSentence : latestReasoningSentence,
      88
    );
    activeModelActionIds.push(actionId);
    activeActions.set(actionId, {
      actionId,
      executionId: execution.id,
      queueIndex: execution.queue_index,
      type: "model",
      toolName: "",
      comparisonName: "",
      primary: modelReasoning !== ""
        ? modelReasoning
        : tr(locale, "conversation.running.primary.modelAnalyzing"),
      secondary: modelReasoning !== ""
        ? tr(locale, "conversation.running.secondary.modelReasoning", { reasoning: modelReasoning })
        : tr(locale, "conversation.running.secondary.modelPending"),
      startedAt: event.timestamp
    });
    return;
  }

  if (event.stage !== "assistant_output" && event.stage !== "turn_limit_reached") {
    return;
  }

  const modelActionId = activeModelActionIds.pop();
  if (modelActionId) {
    activeActions.delete(modelActionId);
  }
}

function handleToolCallEvent(
  execution: Run,
  event: NormalizedTraceEvent,
  locale: TraceLocale,
  activeActions: Map<string, ActiveAction>,
  latestReasoningSentence: string
): void {
  const type: RunningActionType = event.toolName === "run_subagent" ? "subagent" : "tool";
  const toolName = resolveToolName(locale, event.toolName);
  const actionId = event.callId !== ""
    ? `${type}:${execution.id}:${event.callId}`
    : `${type}:${execution.id}:seq:${event.sequence}`;

  activeActions.set(actionId, {
    actionId,
    executionId: execution.id,
    queueIndex: execution.queue_index,
    type,
    toolName,
    comparisonName: resolveToolNameForComparison(event.toolName),
    primary: formatRunningPrimary(locale, type, toolName, event.operationIntentKind, event.operationIntentValue),
    secondary: composeSecondary(locale, latestReasoningSentence, event.operationSummary),
    startedAt: event.timestamp
  });
}

function handleToolResultEvent(
  execution: Run,
  event: NormalizedTraceEvent,
  activeActions: Map<string, ActiveAction>
): void {
  const type: RunningActionType = event.toolName === "run_subagent" ? "subagent" : "tool";
  if (event.callId !== "") {
    activeActions.delete(`${type}:${execution.id}:${event.callId}`);
    return;
  }

  const fallbackCandidate = [...activeActions.values()]
    .filter((item) => item.type === type && item.comparisonName === resolveToolNameForComparison(event.toolName))
    .sort((left, right) => left.startedAt.localeCompare(right.startedAt))[0];
  if (fallbackCandidate) {
    activeActions.delete(fallbackCandidate.actionId);
  }
}

function buildTraceSummary(
  execution: Run,
  events: NormalizedTraceEvent[],
  locale: TraceLocale,
  now: Date
): { primary: string; secondary: string; tone: TraceSummaryTone } {
  const durationSec = resolveDurationSeconds(execution, events, now);
  const messageDurationSec = resolveMessageDurationSeconds(execution, now);
  const thinkingCount = events.filter((event) => event.type === "thinking_delta").length;
  const toolCallCount = events.filter((event) => event.type === "tool_call").length;
  const toolFailedCount = events.filter((event) => event.type === "tool_result" && event.isSuccess === false).length;
  const tokensIn = toOptionalNonNegativeInteger(execution.tokens_in);
  const tokensOut = toOptionalNonNegativeInteger(execution.tokens_out);

  let primary = "";
  if (execution.state === "queued") {
    primary = tr(locale, "conversation.trace.summary.queued");
  } else if (execution.state === "failed") {
    primary = toolFailedCount > 0
      ? tr(locale, "conversation.trace.summary.failed.withCount", {
        tools: toolCallCount,
        failed: toolFailedCount
      })
      : tr(locale, "conversation.trace.summary.failed", { tools: toolCallCount });
  } else if (execution.state === "cancelled") {
    primary = tr(locale, "conversation.trace.summary.cancelled", { tools: toolCallCount });
  } else if (execution.state === "pending" || execution.state === "executing") {
    primary = thinkingCount > 0
      ? tr(locale, "conversation.trace.summary.executing.thinking", {
        duration: durationSec,
        tools: toolCallCount
      })
      : tr(locale, "conversation.trace.summary.executing", {
        duration: durationSec,
        tools: toolCallCount
      });
  } else if (execution.state === "confirming") {
    primary = tr(locale, "conversation.trace.summary.confirming", {
      duration: durationSec,
      tools: toolCallCount
    });
  } else if (execution.state === "awaiting_input") {
    primary = tr(locale, "conversation.trace.summary.awaitingInput", {
      duration: durationSec,
      tools: toolCallCount
    });
  } else {
    primary = tr(locale, "conversation.trace.summary.completed", { tools: toolCallCount });
  }

  const secondary = tokensIn !== null && tokensOut !== null
    ? tr(locale, "conversation.trace.summary.secondary.withToken", {
      input: tokensIn,
      output: tokensOut,
      total: tokensIn + tokensOut,
      duration: messageDurationSec
    })
    : tr(locale, "conversation.trace.summary.secondary.noToken", { duration: messageDurationSec });

  return {
    primary,
    secondary,
    tone: resolveSummaryTone(execution, events)
  };
}

function resolveSummaryTone(execution: Run, events: NormalizedTraceEvent[]): TraceSummaryTone {
  const hasFailedToolResult = events.some((event) => event.type === "tool_result" && event.isSuccess === false);
  if (execution.state === "failed" || hasFailedToolResult) {
    return "error";
  }

  const hasApprovalWaiting = events.some(
    (event) => event.type === "thinking_delta" && event.stage === "run_approval_needed"
  );
  const hasUserQuestionWaiting = events.some(
    (event) => event.type === "thinking_delta" && event.stage === "run_user_question_needed"
  ) && !events.some(
    (event) => event.type === "thinking_delta" && event.stage === "run_user_question_resolved"
  );
  const hasHighRiskToolCall = events.some(
    (event) => event.type === "tool_call" && (event.riskLevel === "high" || event.riskLevel === "critical")
  );
  if (execution.state === "confirming" || execution.state === "awaiting_input" || hasApprovalWaiting || hasUserQuestionWaiting || hasHighRiskToolCall) {
    return "warning";
  }

  if (execution.state === "completed") {
    return "success";
  }
  if (execution.state === "pending" || execution.state === "executing") {
    return "primary";
  }
  if (execution.state === "cancelled" || execution.state === "queued") {
    return "neutral";
  }
  return "primary";
}

function toTraceStep(
  event: NormalizedTraceEvent,
  detailLevel: TraceDetailLevel,
  locale: TraceLocale,
  index: number
): ExecutionTraceStepViewModel {
  const stepId = event.id || `${event.executionId}-${event.sequence}-${index}`;
  const timestampLabel = formatTimestamp(event.timestamp, locale);
  const rawPayload = detailLevel === "verbose" ? event.rawPayload : "";

  if (event.type === "execution_started") {
    return {
      id: stepId,
      kind: "lifecycle",
      title: tr(locale, "conversation.trace.step.title.lifecycle"),
      summary: tr(locale, "conversation.trace.step.summary.executionStarted"),
      detail: "",
      timestampLabel,
      statusTone: "neutral",
      rawPayload
    };
  }

  if (event.type === "thinking_delta") {
    const thinkingSummary = (event.stage === "assistant_output" || event.stage === "model_call") && event.reasoningSentence !== ""
      ? event.reasoningSentence
      : formatThinkingStageLabel(locale, event.stage);
    return {
      id: stepId,
      kind: "reasoning",
      title: tr(locale, "conversation.trace.step.title.reasoning"),
      summary: thinkingSummary,
      detail: resolveThinkingDetail(locale, event),
      timestampLabel,
      statusTone: event.stage === "run_approval_needed" || event.stage === "run_user_question_needed" ? "warning" : "neutral",
      rawPayload
    };
  }

  if (event.type === "tool_call") {
    const toolName = resolveToolName(locale, event.toolName);
    const riskLabel = toRiskLabel(locale, event.riskLevel);
    const toolMeta = riskLabel === ""
      ? tr(locale, "conversation.trace.step.detail.toolMeta", { tool: toolName })
      : tr(locale, "conversation.trace.step.detail.toolMetaWithRisk", { tool: toolName, risk: riskLabel });
    const operationDetail = event.operationSummary === ""
      ? tr(locale, "conversation.trace.step.detail.operationFallback")
      : tr(locale, "conversation.trace.step.detail.operation", {
        operation: event.operationSummary
      });
    return {
      id: stepId,
      kind: "tool_call",
      title: tr(locale, "conversation.trace.step.title.toolCall"),
      summary: formatIntentLabel(locale, event.operationIntentKind, event.operationIntentValue, toolName),
      detail: `${toolMeta} · ${operationDetail}`,
      timestampLabel,
      statusTone: event.riskLevel === "high" || event.riskLevel === "critical" ? "warning" : "neutral",
      rawPayload
    };
  }

  const toolName = resolveToolName(locale, event.toolName);
  const success = event.isSuccess !== false;
  return {
    id: stepId,
    kind: "tool_result",
    title: tr(locale, "conversation.trace.step.title.toolResult"),
    summary: success
      ? tr(locale, "conversation.trace.step.summary.toolResultSuccess", { tool: toolName })
      : tr(locale, "conversation.trace.step.summary.toolResultFailed", { tool: toolName }),
    detail: event.resultSummary !== ""
      ? event.resultSummary
      : success
        ? tr(locale, "conversation.trace.step.detail.resultFallbackSuccess")
        : tr(locale, "conversation.trace.step.detail.resultFallbackFailed"),
    timestampLabel,
    statusTone: success ? "success" : "error",
    rawPayload
  };
}

function resolveThinkingDetail(locale: TraceLocale, event: NormalizedTraceEvent): string {
  if (event.stage === "run_approval_needed") {
    if (event.operationSummary !== "") {
      return tr(locale, "conversation.trace.step.detail.operation", { operation: event.operationSummary });
    }
    return tr(locale, "conversation.trace.step.detail.waitingApproval");
  }

  if (event.stage === "run_user_question_needed") {
    const question = String(event.payload.question ?? "").trim();
    if (question !== "") {
      return tr(locale, "conversation.trace.step.detail.userQuestion", { question });
    }
    return tr(locale, "conversation.trace.step.detail.waitingUserInput");
  }

  if (event.stage === "assistant_output" || event.stage === "model_call") {
    return "";
  }

  if (event.reasoningSentence !== "") {
    return event.reasoningSentence;
  }
  return "";
}

function formatThinkingStageLabel(locale: TraceLocale, stage: NormalizedThinkingStage): string {
  switch (stage) {
    case "model_call":
      return tr(locale, "conversation.trace.stage.modelCall");
    case "assistant_output":
      return tr(locale, "conversation.trace.stage.assistantOutput");
    case "run_approval_needed":
      return tr(locale, "conversation.trace.stage.approvalNeeded");
    case "run_user_question_needed":
      return tr(locale, "conversation.trace.stage.userQuestionNeeded");
    case "run_user_question_resolved":
      return tr(locale, "conversation.trace.stage.userQuestionResolved");
    case "approval_granted":
      return tr(locale, "conversation.trace.stage.approvalGranted");
    case "approval_denied":
      return tr(locale, "conversation.trace.stage.approvalDenied");
    case "approval_resolved":
      return tr(locale, "conversation.trace.stage.approvalResolved");
    case "turn_limit_reached":
      return tr(locale, "conversation.trace.stage.turnLimitReached");
    default:
      return tr(locale, "conversation.trace.stage.other");
  }
}

function toRiskLabel(locale: TraceLocale, riskLevel: string): string {
  const normalized = riskLevel.trim().toLowerCase();
  if (normalized === "critical") {
    return tr(locale, "conversation.trace.risk.critical");
  }
  if (normalized === "high") {
    return tr(locale, "conversation.trace.risk.high");
  }
  if (normalized === "low") {
    return tr(locale, "conversation.trace.risk.low");
  }
  return "";
}

function formatIntentLabel(
  locale: TraceLocale,
  intentKind: OperationIntentKind,
  intentValue: string,
  toolName: string
): string {
  const value = truncateText(intentValue, 120);

  if (intentKind === "command" && value !== "") {
    return tr(locale, "conversation.trace.intent.command", { value });
  }
  if (intentKind === "path" && value !== "") {
    return tr(locale, "conversation.trace.intent.path", { value });
  }
  if (intentKind === "url" && value !== "") {
    return tr(locale, "conversation.trace.intent.url", { value });
  }
  if (intentKind === "query" && value !== "") {
    return tr(locale, "conversation.trace.intent.query", { value });
  }
  if (intentKind === "scalar" && value !== "") {
    return tr(locale, "conversation.trace.intent.scalar", { value });
  }
  return tr(locale, "conversation.trace.intent.toolFallback", { tool: toolName });
}

function isMeaningfulTraceStepEvent(event: NormalizedTraceEvent): boolean {
  if (event.type !== "thinking_delta") {
    return true;
  }
  return isMeaningfulThinkingEvent(event);
}

function isMeaningfulThinkingEvent(event: NormalizedTraceEvent): boolean {
  if (event.type !== "thinking_delta") {
    return true;
  }

  if (event.stage === "model_call" || event.stage === "assistant_output") {
    return event.reasoningSentence !== "";
  }
  if (event.stage === "run_approval_needed") {
    return true;
  }
  if (event.stage === "run_user_question_needed" || event.stage === "run_user_question_resolved") {
    return true;
  }
  if (event.stage === "approval_granted" || event.stage === "approval_denied" || event.stage === "approval_resolved") {
    return true;
  }
  if (event.stage === "turn_limit_reached") {
    return true;
  }
  return event.reasoningSentence !== "";
}

function formatRunningPrimary(
  locale: TraceLocale,
  type: RunningActionType,
  toolName: string,
  operationIntentKind: OperationIntentKind,
  operationIntentValue: string
): string {
  if (type === "model") {
    return tr(locale, "conversation.running.primary.modelAnalyzing");
  }
  if (type === "approval") {
    return tr(locale, "conversation.running.primary.approval", { tool: toolName });
  }
  if (type === "user_input") {
    return tr(locale, "conversation.running.primary.userInput");
  }
  return formatIntentLabel(locale, operationIntentKind, operationIntentValue, toolName);
}

function composeSecondary(locale: TraceLocale, reasoning: string, operation: string): string {
  const normalizedReasoning = truncateText(reasoning, 88);
  const normalizedOperation = truncateText(operation, 120);

  if (normalizedReasoning !== "" && normalizedOperation !== "") {
    return tr(locale, "conversation.running.secondary.reasoningOperation", {
      reasoning: normalizedReasoning,
      operation: normalizedOperation
    });
  }
  if (normalizedReasoning !== "") {
    return tr(locale, "conversation.running.secondary.reasoningOnly", {
      reasoning: normalizedReasoning
    });
  }
  if (normalizedOperation !== "") {
    return tr(locale, "conversation.running.secondary.operationOnly", {
      operation: normalizedOperation
    });
  }
  return tr(locale, "conversation.running.secondary.pending");
}

function resolveToolName(locale: TraceLocale, toolName: string): string {
  const normalized = toolName.trim();
  if (normalized === "") {
    return tr(locale, "conversation.trace.toolFallback");
  }
  return normalized;
}

function resolveToolNameForComparison(toolName: string): string {
  return toolName.trim() || "tool";
}

function formatTimestamp(value: string, locale: TraceLocale): string {
  const date = toDateOrNow(value);
  try {
    return new Intl.DateTimeFormat(locale, {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false
    }).format(date);
  } catch {
    return "";
  }
}

function resolveDurationSeconds(execution: Run, events: NormalizedTraceEvent[], now: Date): number {
  const startedAt = resolveStartedAt(execution, events);
  const endedAt = resolveEndedAt(execution, now);
  const durationMs = endedAt.getTime() - startedAt.getTime();
  if (!Number.isFinite(durationMs) || durationMs <= 0) {
    return 0;
  }
  return Math.round(durationMs / 1000);
}

function resolveMessageDurationSeconds(execution: Run, now: Date): number {
  const startedAt = toDateOrNow(execution.created_at);
  const endedAt = resolveEndedAt(execution, now);
  const durationMs = endedAt.getTime() - startedAt.getTime();
  if (!Number.isFinite(durationMs) || durationMs <= 0) {
    return 0;
  }
  return Math.round(durationMs / 1000);
}

function resolveStartedAt(execution: Run, events: NormalizedTraceEvent[]): Date {
  const startedEvent = events.find((item) => item.type === "execution_started");
  return toDateOrNow(startedEvent?.timestamp || execution.created_at);
}

function resolveEndedAt(execution: Run, now: Date): Date {
  if (!isTerminalRunState(execution.state)) {
    return now;
  }
  return toDateOrNow(execution.updated_at);
}

function toDateOrNow(value: string): Date {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return new Date();
  }
  return parsed;
}

function toOptionalNonNegativeInteger(value: unknown): number | null {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return null;
  }
  if (value < 0) {
    return 0;
  }
  return Math.trunc(value);
}

function tr(locale: TraceLocale, key: string, params: Record<string, string | number> = {}): string {
  const template = messages[locale][key] ?? messages["zh-CN"][key] ?? key;
  return template.replace(/\{(\w+)\}/g, (_, match) => {
    if (!(match in params)) {
      return `{${match}}`;
    }
    return String(params[match]);
  });
}
