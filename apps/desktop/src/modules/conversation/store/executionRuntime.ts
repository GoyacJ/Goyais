import { createMockId } from "@/shared/utils/id";
import type { ConversationSnapshot, DiffItem, Execution, ExecutionEvent } from "@/shared/types/api";

import {
  cloneExecution,
  mergeExecution,
  normalizeExecutionList,
  resolveMergedExecutionState
} from "@/modules/conversation/store/executionMerge";
import type { ConversationRuntime } from "@/modules/conversation/store/state";

export function ensureExecution(runtime: ConversationRuntime, conversationId: string, event: ExecutionEvent): Execution {
  const executionId = event.execution_id.trim();
  let execution = runtime.executions.find((item) => item.id === executionId);
  if (execution) {
    return execution;
  }

  execution = upsertExecutionFromServer(runtime, {
    id: executionId,
    workspace_id: "",
    conversation_id: conversationId,
    message_id: "",
    state: "queued",
    mode: "agent",
    model_id: "",
    mode_snapshot: "agent",
    model_snapshot: {
      model_id: ""
    },
    project_revision_snapshot: 0,
    queue_index: event.queue_index,
    trace_id: event.trace_id,
    created_at: event.timestamp,
    updated_at: event.timestamp
  });
  return execution;
}

export function applyExecutionState(execution: Execution, event: ExecutionEvent): void {
  const nextState = executionStateByEventType[event.type];
  if (nextState) {
    execution.state = resolveMergedExecutionState(execution.state, nextState);
  }
  const usage = parseUsageFromPayload(event.payload);
  if (usage) {
    execution.tokens_in = Math.max(execution.tokens_in ?? 0, usage.inputTokens);
    execution.tokens_out = Math.max(execution.tokens_out ?? 0, usage.outputTokens);
  }
  execution.updated_at = event.timestamp;
}

const executionStateByEventType: Partial<Record<ExecutionEvent["type"], Execution["state"]>> = {
  execution_started: "executing",
  execution_stopped: "cancelled",
  execution_done: "completed",
  execution_error: "failed"
};

function parseUsageFromPayload(payload: Record<string, unknown>): { inputTokens: number; outputTokens: number } | null {
  const usage = payload.usage;
  if (!usage || typeof usage !== "object") {
    return null;
  }
  const usageMap = usage as Record<string, unknown>;
  const inputTokens = toNonNegativeInteger(usageMap.input_tokens);
  const outputTokens = toNonNegativeInteger(usageMap.output_tokens);
  if (inputTokens === null && outputTokens === null) {
    return null;
  }
  return {
    inputTokens: inputTokens ?? 0,
    outputTokens: outputTokens ?? 0
  };
}

function toNonNegativeInteger(value: unknown): number | null {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return null;
  }
  if (value < 0) {
    return 0;
  }
  return Math.trunc(value);
}

export function upsertExecutionFromServer(runtime: ConversationRuntime, incoming: Execution): Execution {
  const normalizedIncoming = cloneExecution(incoming);
  const index = runtime.executions.findIndex((item) => item.id === normalizedIncoming.id);
  if (index < 0) {
    runtime.executions.push(normalizedIncoming);
    return normalizedIncoming;
  }

  const merged = mergeExecution(runtime.executions[index], normalizedIncoming);
  runtime.executions[index] = merged;
  return merged;
}

export function dedupeExecutions(runtime: ConversationRuntime): void {
  runtime.executions = normalizeExecutionList(runtime.executions);
}

export function parseDiff(payload: Record<string, unknown>): DiffItem[] {
  const raw = payload.diff;
  if (!Array.isArray(raw)) {
    return [];
  }

  return raw
    .filter((item): item is Record<string, unknown> => typeof item === "object" && item !== null)
    .map((item) => ({
      id: typeof item.id === "string" ? item.id : createMockId("diff"),
      path: typeof item.path === "string" ? item.path : "unknown",
      change_type: item.change_type === "added" || item.change_type === "deleted" ? item.change_type : "modified",
      summary: typeof item.summary === "string" ? item.summary : "changed"
    }));
}

export function restoreExecutionsFromSnapshot(
  runtime: ConversationRuntime,
  conversationId: string,
  snapshot: ConversationSnapshot
): Execution[] {
  const existingById = new Map(runtime.executions.map((execution) => [execution.id, execution]));
  if (Array.isArray(snapshot.execution_snapshots) && snapshot.execution_snapshots.length > 0) {
    return snapshot.execution_snapshots.map((item) => {
      const existing = existingById.get(item.id);
      if (existing) {
        return {
          ...existing,
          state: item.state,
          queue_index: item.queue_index,
          message_id: item.message_id,
          updated_at: item.updated_at
        };
      }

      const timestamp = item.updated_at || new Date().toISOString();
      return {
        id: item.id,
        workspace_id: "",
        conversation_id: conversationId,
        message_id: item.message_id,
        state: item.state,
        mode: "agent",
        model_id: "",
        mode_snapshot: "agent",
        model_snapshot: {
          model_id: ""
        },
        project_revision_snapshot: 0,
        queue_index: item.queue_index,
        trace_id: "",
        created_at: timestamp,
        updated_at: timestamp
      };
    });
  }

  // Backward-compatibility for snapshots created before execution_snapshots was introduced.
  return runtime.executions.filter((execution) => snapshot.execution_ids.includes(execution.id));
}
