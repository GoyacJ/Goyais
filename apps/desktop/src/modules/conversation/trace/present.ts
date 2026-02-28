import { isTerminalExecutionState } from "@/modules/conversation/store/executionMerge";
import { normalizeExecutionEventsByExecution } from "@/modules/conversation/trace/normalize";
import { truncateText } from "@/modules/conversation/trace/summarize";
import type {
  ExecutionTraceStepViewModel,
  ExecutionTraceViewModelData,
  NormalizedThinkingStage,
  NormalizedTraceEvent,
  RunningActionBaseViewModel,
  RunningActionType,
  RunningActionViewModelData,
  TraceLocale,
  TraceStatusTone
} from "@/modules/conversation/trace/types";
import { messages } from "@/shared/i18n/messages";
import type { Execution, ExecutionEvent, TraceDetailLevel } from "@/shared/types/api";

type ActiveAction = RunningActionBaseViewModel & {
  toolName: string;
  comparisonName: string;
};

export function buildExecutionTraceViewModelData(
  events: ExecutionEvent[],
  executions: Execution[],
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

      return {
        executionId: execution.id,
        messageId: execution.message_id,
        queueIndex: execution.queue_index,
        state: execution.state,
        isRunning: execution.state === "pending" || execution.state === "executing" || execution.state === "confirming",
        summaryPrimary: summary.primary,
        summarySecondary: summary.secondary,
        steps: normalizedEvents.map((event, index) => toTraceStep(event, detailLevel, locale, index))
      };
    });
}

export function buildRunningActionBaseViewModelData(
  events: ExecutionEvent[],
  executions: Execution[],
  locale: TraceLocale
): RunningActionBaseViewModel[] {
  const groupedEvents = normalizeExecutionEventsByExecution(events);
  const runningExecutions = executions
    .filter((execution) => execution.state === "pending" || execution.state === "executing" || execution.state === "confirming")
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
  execution: Execution,
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
  execution: Execution,
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

  if (event.stage === "approval_granted" || event.stage === "approval_denied" || event.stage === "approval_resolved") {
    for (const action of [...activeActions.values()]) {
      if (action.type === "approval") {
        activeActions.delete(action.actionId);
      }
    }
    return;
  }

  if (event.stage === "model_call") {
    const actionId = `model:${execution.id}:${event.sequence}`;
    activeModelActionIds.push(actionId);
    activeActions.set(actionId, {
      actionId,
      executionId: execution.id,
      queueIndex: execution.queue_index,
      type: "model",
      toolName: "",
      comparisonName: "",
      primary: tr(locale, "conversation.running.primary.model"),
      secondary: composeSecondary(locale, event.reasoningSentence, ""),
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
  execution: Execution,
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
    primary: formatRunningPrimary(locale, type, toolName),
    secondary: composeSecondary(locale, latestReasoningSentence, event.operationSummary),
    startedAt: event.timestamp
  });
}

function handleToolResultEvent(
  execution: Execution,
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
  execution: Execution,
  events: NormalizedTraceEvent[],
  locale: TraceLocale,
  now: Date
): { primary: string; secondary: string } {
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

  return { primary, secondary };
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
    return {
      id: stepId,
      kind: "reasoning",
      title: tr(locale, "conversation.trace.step.title.reasoning"),
      summary: formatThinkingStageLabel(locale, event.stage),
      detail: resolveThinkingDetail(locale, event),
      timestampLabel,
      statusTone: event.stage === "run_approval_needed" ? "warning" : "neutral",
      rawPayload
    };
  }

  if (event.type === "tool_call") {
    const toolName = resolveToolName(locale, event.toolName);
    const riskLabel = toRiskLabel(locale, event.riskLevel);
    return {
      id: stepId,
      kind: "tool_call",
      title: tr(locale, "conversation.trace.step.title.toolCall"),
      summary: riskLabel === ""
        ? tr(locale, "conversation.trace.step.summary.toolCall", { tool: toolName })
        : tr(locale, "conversation.trace.step.summary.toolCallWithRisk", { tool: toolName, risk: riskLabel }),
      detail: event.operationSummary === ""
        ? tr(locale, "conversation.trace.step.detail.operationFallback")
        : tr(locale, "conversation.trace.step.detail.operation", {
          operation: event.operationSummary
        }),
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
  if (event.reasoningSentence !== "" && event.reasoningSentence !== "thinking") {
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

function formatRunningPrimary(locale: TraceLocale, type: RunningActionType, toolName: string): string {
  if (type === "model") {
    return tr(locale, "conversation.running.primary.model");
  }
  if (type === "approval") {
    return tr(locale, "conversation.running.primary.approval", { tool: toolName });
  }
  if (type === "subagent") {
    return tr(locale, "conversation.running.primary.subagent", { tool: toolName });
  }
  return tr(locale, "conversation.running.primary.tool", { tool: toolName });
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

function resolveDurationSeconds(execution: Execution, events: NormalizedTraceEvent[], now: Date): number {
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

function resolveStartedAt(execution: Execution, events: NormalizedTraceEvent[]): Date {
  const startedEvent = events.find((item) => item.type === "execution_started");
  return toDateOrNow(startedEvent?.timestamp || execution.created_at);
}

function resolveEndedAt(execution: Execution, now: Date): Date {
  if (!isTerminalExecutionState(execution.state)) {
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
