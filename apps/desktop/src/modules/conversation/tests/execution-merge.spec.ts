import { describe, expect, it } from "vitest";

import { normalizeExecutionList } from "@/modules/conversation/store/executionMerge";
import type { Execution } from "@/shared/types/api";

describe("execution merge", () => {
  it("dedupes same execution id and keeps terminal state", () => {
    const duplicated: Execution[] = [
      buildExecution({
        id: "exec_1",
        state: "completed"
      }),
      buildExecution({
        id: "exec_1",
        state: "pending"
      })
    ];

    const normalized = normalizeExecutionList(duplicated);
    expect(normalized).toHaveLength(1);
    expect(normalized[0]?.id).toBe("exec_1");
    expect(normalized[0]?.state).toBe("completed");
  });
});

function buildExecution(
  partial: Partial<Execution> & Pick<Execution, "id" | "state">
): Execution {
  const now = new Date().toISOString();
  return {
    id: partial.id,
    workspace_id: partial.workspace_id ?? "ws_local",
    conversation_id: partial.conversation_id ?? "conv_1",
    message_id: partial.message_id ?? "msg_1",
    state: partial.state,
    mode: partial.mode ?? "agent",
    model_id: partial.model_id ?? "gpt-5.3",
    mode_snapshot: partial.mode_snapshot ?? "agent",
    model_snapshot: partial.model_snapshot ?? { model_id: "gpt-5.3" },
    project_revision_snapshot: partial.project_revision_snapshot ?? 0,
    queue_index: partial.queue_index ?? 0,
    trace_id: partial.trace_id ?? "tr_1",
    created_at: partial.created_at ?? now,
    updated_at: partial.updated_at ?? now
  };
}
