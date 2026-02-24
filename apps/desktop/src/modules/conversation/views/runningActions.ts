import type { Execution, ExecutionEvent } from "@/shared/types/api";

export type RunningActionType = "model" | "tool" | "subagent";

export type RunningActionViewModel = {
  actionId: string;
  executionId: string;
  queueIndex: number;
  type: RunningActionType;
  label: string;
  startedAt: string;
  elapsedMs: number;
  elapsedLabel: string;
};

type ActiveAction = {
  actionId: string;
  executionId: string;
  queueIndex: number;
  type: RunningActionType;
  name: string;
  startedAt: string;
};

const RENDERED_EVENT_TYPES: ReadonlySet<string> = new Set([
  "execution_started",
  "thinking_delta",
  "tool_call",
  "tool_result"
]);

export function buildRunningActionViewModels(
  events: ExecutionEvent[],
  executions: Execution[],
  now: Date = new Date()
): RunningActionViewModel[] {
  const groupedEvents = groupExecutionEvents(events);
  const runningExecutions = executions
    .filter((execution) => execution.state === "pending" || execution.state === "executing")
    .sort((left, right) => {
      if (left.queue_index !== right.queue_index) {
        return left.queue_index - right.queue_index;
      }
      return left.created_at.localeCompare(right.created_at);
    });

  const activeActions = runningExecutions.flatMap((execution) =>
    collectActiveActionsForExecution(execution, groupedEvents.get(execution.id) ?? [])
  );

  return activeActions
    .sort((left, right) => {
      if (left.queueIndex !== right.queueIndex) {
        return left.queueIndex - right.queueIndex;
      }
      return left.startedAt.localeCompare(right.startedAt);
    })
    .map((action) => {
      const elapsedMs = Math.max(0, now.getTime() - toDateOrNow(action.startedAt).getTime());
      return {
        actionId: action.actionId,
        executionId: action.executionId,
        queueIndex: action.queueIndex,
        type: action.type,
        label: toRunningActionLabel(action),
        startedAt: action.startedAt,
        elapsedMs,
        elapsedLabel: `${Math.floor(elapsedMs / 1000)}s`
      };
    });
}

function groupExecutionEvents(events: ExecutionEvent[]): Map<string, ExecutionEvent[]> {
  const grouped = new Map<string, ExecutionEvent[]>();
  for (const event of events) {
    if (!RENDERED_EVENT_TYPES.has(event.type)) {
      continue;
    }
    const executionId = event.execution_id.trim();
    if (executionId === "") {
      continue;
    }
    const list = grouped.get(executionId) ?? [];
    list.push(event);
    grouped.set(executionId, list);
  }
  for (const [executionId, list] of grouped.entries()) {
    grouped.set(executionId, sortEvents(list));
  }
  return grouped;
}

function sortEvents(events: ExecutionEvent[]): ExecutionEvent[] {
  return [...events].sort((left, right) => {
    if (left.sequence !== right.sequence) {
      return left.sequence - right.sequence;
    }
    return left.timestamp.localeCompare(right.timestamp);
  });
}

function collectActiveActionsForExecution(execution: Execution, events: ExecutionEvent[]): ActiveAction[] {
  const activeActions = new Map<string, ActiveAction>();
  const activeModelActionIds: string[] = [];

  for (const event of events) {
    if (event.type === "thinking_delta") {
      const stage = asString(event.payload.stage);
      if (stage === "model_call") {
        const actionId = `model:${execution.id}:${event.sequence}`;
        activeModelActionIds.push(actionId);
        activeActions.set(actionId, {
          actionId,
          executionId: execution.id,
          queueIndex: execution.queue_index,
          type: "model",
          name: "model_call",
          startedAt: event.timestamp
        });
      } else if (stage === "assistant_output" || stage === "turn_limit_reached") {
        const modelActionId = activeModelActionIds.pop();
        if (modelActionId) {
          activeActions.delete(modelActionId);
        }
      }
      continue;
    }

    if (event.type === "tool_call") {
      const name = asString(event.payload.name) || "tool";
      const actionType: RunningActionType = name === "run_subagent" ? "subagent" : "tool";
      const callID = asString(event.payload.call_id);
      const actionId = callID !== ""
        ? `${actionType}:${execution.id}:${callID}`
        : `${actionType}:${execution.id}:seq:${event.sequence}`;
      activeActions.set(actionId, {
        actionId,
        executionId: execution.id,
        queueIndex: execution.queue_index,
        type: actionType,
        name,
        startedAt: event.timestamp
      });
      continue;
    }

    if (event.type === "tool_result") {
      const name = asString(event.payload.name) || "tool";
      const actionType: RunningActionType = name === "run_subagent" ? "subagent" : "tool";
      const callID = asString(event.payload.call_id);
      if (callID !== "") {
        activeActions.delete(`${actionType}:${execution.id}:${callID}`);
        continue;
      }
      const fallbackCandidate = [...activeActions.values()]
        .filter((item) => item.type === actionType && item.name === name)
        .sort((left, right) => left.startedAt.localeCompare(right.startedAt))[0];
      if (fallbackCandidate) {
        activeActions.delete(fallbackCandidate.actionId);
      }
    }
  }
  return [...activeActions.values()];
}

function toDateOrNow(input: string): Date {
  const value = new Date(input);
  if (Number.isNaN(value.getTime())) {
    return new Date();
  }
  return value;
}

function asString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function toRunningActionLabel(action: ActiveAction): string {
  if (action.type === "model") {
    return "模型推理";
  }
  if (action.type === "subagent") {
    return `子代理 ${action.name}`;
  }
  return `工具 ${action.name}`;
}
