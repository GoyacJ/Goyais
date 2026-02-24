import { beforeEach, describe, expect, it } from "vitest";

import {
  ensureConversationRuntime,
  getExecutionStateCounts,
  hydrateConversationRuntime,
  resetConversationStore
} from "@/modules/conversation/store";
import type { Conversation } from "@/shared/types/api";

const mockConversation: Conversation = {
  id: "conv_test",
  workspace_id: "ws_local",
  project_id: "proj_1",
  name: "Test Conversation",
  queue_state: "idle",
  default_mode: "agent",
  model_id: "gpt-5.3",
  base_revision: 0,
  active_execution_id: null,
  created_at: "2026-02-23T00:00:00Z",
  updated_at: "2026-02-23T00:00:00Z"
};

describe("conversation runtime hydration", () => {
  beforeEach(() => {
    resetConversationStore();
  });

  it("hydrates runtime from conversation detail response", () => {
    const detail = {
      conversation: {
        ...mockConversation,
        default_mode: "plan" as const,
        model_id: "MiniMax-M2.5"
      },
      messages: [
        {
          id: "msg_seed_1",
          conversation_id: mockConversation.id,
          role: "user" as const,
          content: "查看当前项目",
          created_at: "2026-02-24T00:00:00Z"
        }
      ],
      executions: [
        {
          id: "exec_seed_1",
          workspace_id: "ws_local",
          conversation_id: mockConversation.id,
          message_id: "msg_seed_1",
          state: "pending" as const,
          mode: "plan" as const,
          model_id: "MiniMax-M2.5",
          mode_snapshot: "plan" as const,
          model_snapshot: {
            model_id: "MiniMax-M2.5"
          },
          project_revision_snapshot: 0,
          queue_index: 0,
          trace_id: "tr_seed_1",
          created_at: "2026-02-24T00:00:00Z",
          updated_at: "2026-02-24T00:00:00Z"
        }
      ],
      snapshots: []
    };

    const runtime = hydrateConversationRuntime(mockConversation, true, detail);
    expect(runtime.mode).toBe("plan");
    expect(runtime.modelId).toBe("MiniMax-M2.5");
    expect(runtime.messages.length).toBe(1);
    expect(runtime.messages[0]?.content).toBe("查看当前项目");
    expect(runtime.executions.length).toBe(1);
    expect(runtime.executions[0]?.state).toBe("pending");
  });

  it("counts pending/executing/queued states", () => {
    const runtime = ensureConversationRuntime(mockConversation, true);
    runtime.executions = [
      {
        id: "exec_pending",
        workspace_id: "ws_local",
        conversation_id: mockConversation.id,
        message_id: "msg_pending",
        state: "pending",
        mode: "agent",
        model_id: "gpt-5.3",
        mode_snapshot: "agent",
        model_snapshot: { model_id: "gpt-5.3" },
        project_revision_snapshot: 0,
        queue_index: 0,
        trace_id: "tr_pending",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      },
      {
        id: "exec_executing",
        workspace_id: "ws_local",
        conversation_id: mockConversation.id,
        message_id: "msg_executing",
        state: "executing",
        mode: "agent",
        model_id: "gpt-5.3",
        mode_snapshot: "agent",
        model_snapshot: { model_id: "gpt-5.3" },
        project_revision_snapshot: 0,
        queue_index: 1,
        trace_id: "tr_executing",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      },
      {
        id: "exec_queued",
        workspace_id: "ws_local",
        conversation_id: mockConversation.id,
        message_id: "msg_queued",
        state: "queued",
        mode: "agent",
        model_id: "gpt-5.3",
        mode_snapshot: "agent",
        model_snapshot: { model_id: "gpt-5.3" },
        project_revision_snapshot: 0,
        queue_index: 2,
        trace_id: "tr_queued",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      }
    ];

    const counts = getExecutionStateCounts(runtime);
    expect(counts.pending).toBe(1);
    expect(counts.executing).toBe(1);
    expect(counts.queued).toBe(1);
  });
});
