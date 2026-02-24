import { createMockId } from "@/shared/services/mockData";
import type { ConversationSnapshot, DiffItem, Execution, ExecutionEvent } from "@/shared/types/api";

import type { ConversationRuntime } from "@/modules/conversation/store/state";

export function ensureExecution(runtime: ConversationRuntime, conversationId: string, event: ExecutionEvent): Execution {
  let execution = runtime.executions.find((item) => item.id === event.execution_id);
  if (execution) {
    return execution;
  }

  execution = {
    id: event.execution_id,
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
  };
  runtime.executions.push(execution);
  return execution;
}

export function applyExecutionState(execution: Execution, event: ExecutionEvent): void {
  if (event.type === "confirmation_resolved") {
    const decision = typeof event.payload.decision === "string" ? event.payload.decision.toLowerCase() : "";
    execution.state = decision === "deny" ? "cancelled" : "executing";
  } else {
    const nextState = executionStateByEventType[event.type];
    if (nextState) {
      execution.state = nextState;
    }
  }
  execution.updated_at = event.timestamp;
}

const executionStateByEventType: Partial<Record<ExecutionEvent["type"], Execution["state"]>> = {
  execution_started: "executing",
  confirmation_required: "confirming",
  execution_stopped: "cancelled",
  execution_done: "completed",
  execution_error: "failed"
};

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
