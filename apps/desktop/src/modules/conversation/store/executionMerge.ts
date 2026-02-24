import type { Execution } from "@/shared/types/api";

const executionStateRank: Record<Execution["state"], number> = {
  queued: 0,
  pending: 1,
  executing: 2,
  cancelled: 3,
  failed: 4,
  completed: 5
};

export function cloneExecution(execution: Execution): Execution {
  return {
    ...execution,
    id: execution.id.trim(),
    model_snapshot: {
      ...execution.model_snapshot
    },
    agent_config_snapshot: execution.agent_config_snapshot
      ? { ...execution.agent_config_snapshot }
      : undefined
  };
}

export function isTerminalExecutionState(state: Execution["state"]): boolean {
  return state === "completed" || state === "failed" || state === "cancelled";
}

export function resolveMergedExecutionState(
  current: Execution["state"],
  incoming: Execution["state"]
): Execution["state"] {
  if (current === incoming) {
    return current;
  }
  const currentTerminal = isTerminalExecutionState(current);
  const incomingTerminal = isTerminalExecutionState(incoming);
  if (currentTerminal && !incomingTerminal) {
    return current;
  }
  if (!currentTerminal && incomingTerminal) {
    return incoming;
  }
  return executionStateRank[incoming] >= executionStateRank[current] ? incoming : current;
}

export function mergeExecution(current: Execution, incoming: Execution): Execution {
  return {
    ...current,
    ...incoming,
    state: resolveMergedExecutionState(current.state, incoming.state),
    workspace_id: preferNonEmpty(current.workspace_id, incoming.workspace_id),
    conversation_id: preferNonEmpty(current.conversation_id, incoming.conversation_id),
    message_id: preferNonEmpty(current.message_id, incoming.message_id),
    model_id: preferNonEmpty(current.model_id, incoming.model_id),
    trace_id: preferNonEmpty(current.trace_id, incoming.trace_id),
    created_at: preferEarlierTimestamp(current.created_at, incoming.created_at),
    updated_at: preferLaterTimestamp(current.updated_at, incoming.updated_at),
    model_snapshot: {
      ...current.model_snapshot,
      ...incoming.model_snapshot
    },
    agent_config_snapshot: incoming.agent_config_snapshot
      ? { ...incoming.agent_config_snapshot }
      : current.agent_config_snapshot
        ? { ...current.agent_config_snapshot }
        : undefined
  };
}

export function normalizeExecutionList(executions: Execution[]): Execution[] {
  if (executions.length <= 1) {
    return executions;
  }

  const executionByID = new Map<string, Execution>();
  const order: string[] = [];
  for (const execution of executions) {
    const normalized = cloneExecution(execution);
    const existing = executionByID.get(normalized.id);
    if (!existing) {
      executionByID.set(normalized.id, normalized);
      order.push(normalized.id);
      continue;
    }
    executionByID.set(normalized.id, mergeExecution(existing, normalized));
  }

  return order
    .map((executionID) => executionByID.get(executionID))
    .filter((execution): execution is Execution => Boolean(execution));
}

function preferNonEmpty(current: string, incoming: string): string {
  const normalizedIncoming = incoming.trim();
  if (normalizedIncoming !== "") {
    return normalizedIncoming;
  }
  return current;
}

function preferEarlierTimestamp(current: string, incoming: string): string {
  if (incoming.trim() === "") {
    return current;
  }
  if (current.trim() === "" || incoming < current) {
    return incoming;
  }
  return current;
}

function preferLaterTimestamp(current: string, incoming: string): string {
  if (incoming.trim() === "") {
    return current;
  }
  if (current.trim() === "" || incoming > current) {
    return incoming;
  }
  return current;
}
