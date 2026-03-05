import { ref } from "vue";
import { describe, expect, it } from "vitest";

import type { ConversationRuntime } from "@/modules/conversation/store/state";
import { useQueueMessagesView } from "@/modules/conversation/views/useQueueMessagesView";

describe("useQueueMessagesView", () => {
  it("derives FIFO queued messages and hides queued user messages in main list", () => {
    const runtime = ref(
      createRuntime({
        messages: [
          createUserMessage("msg_0", 0, "当前执行消息"),
          createUserMessage("msg_1", 1, "队列消息一"),
          createUserMessage("msg_2", 2, "队列消息二"),
          createAssistantMessage("msg_a", "当前输出")
        ],
        executions: [
          createExecution("exec_0", 0, "pending"),
          createExecution("exec_1", 1, "queued"),
          createExecution("exec_2", 2, "queued")
        ]
      })
    );

    const { queuedMessages, visibleMessages, visibleTraceExecutionIds } = useQueueMessagesView(runtime);

    expect(queuedMessages.value.map((item) => item.queueIndex)).toEqual([1, 2]);
    expect(queuedMessages.value.map((item) => item.content)).toEqual(["队列消息一", "队列消息二"]);
    expect(visibleMessages.value.map((item) => item.id)).toEqual(["msg_0", "msg_a"]);
    expect(visibleTraceExecutionIds.value.has("exec_0")).toBe(true);
    expect(visibleTraceExecutionIds.value.has("exec_1")).toBe(false);
    expect(visibleTraceExecutionIds.value.has("exec_2")).toBe(false);
  });

  it("hides cancelled queued message removed via run control stop", () => {
    const runtime = ref(
      createRuntime({
        messages: [
          createUserMessage("msg_0", 0, "主区消息"),
          createUserMessage("msg_1", 1, "要移除的队列消息")
        ],
        executions: [
          createExecution("exec_0", 0, "executing"),
          createExecution("exec_1", 1, "cancelled")
        ],
        events: [
          {
            event_id: "evt_remove_1",
            execution_id: "exec_1",
            conversation_id: "conv_1",
            trace_id: "tr_1",
            sequence: 1,
            queue_index: 1,
            type: "execution_stopped",
            timestamp: "2026-02-28T10:00:00Z",
            payload: {
              action: "stop",
              source: "run_control"
            }
          }
        ]
      })
    );

    const { visibleMessages, visibleTraceExecutionIds } = useQueueMessagesView(runtime);

    expect(visibleMessages.value.map((item) => item.id)).toEqual(["msg_0"]);
    expect(visibleTraceExecutionIds.value.has("exec_1")).toBe(false);
  });

  it("keeps current message visible when cancelled by conversation stop", () => {
    const runtime = ref(
      createRuntime({
        messages: [createUserMessage("msg_0", 0, "被停止但应保留")],
        executions: [createExecution("exec_0", 0, "cancelled")],
        events: [
          {
            event_id: "evt_stop_current",
            execution_id: "exec_0",
            conversation_id: "conv_1",
            trace_id: "tr_current",
            sequence: 1,
            queue_index: 0,
            type: "execution_stopped",
            timestamp: "2026-02-28T10:00:00Z",
            payload: {
              reason: "user_stop"
            }
          }
        ]
      })
    );

    const { visibleMessages, visibleTraceExecutionIds } = useQueueMessagesView(runtime);

    expect(visibleMessages.value.map((item) => item.id)).toEqual(["msg_0"]);
    expect(visibleTraceExecutionIds.value.has("exec_0")).toBe(true);
  });
});

function createRuntime(overrides: Partial<ConversationRuntime>): ConversationRuntime {
  const runtime: ConversationRuntime = {
    messages: [],
    events: [],
    runs: [],
    executions: [],
    snapshots: [],
    draft: "",
    mode: "default",
    modelId: "rc_model_1",
    ruleIds: [],
    skillIds: [],
    mcpIds: [],
    status: "connected",
    diff: [],
    projectKind: "git",
    diffCapability: {
      can_commit: true,
      can_discard: true,
      can_export: true,
      can_export_patch: true
    },
    changeSet: null,
    inspectorTab: "diff",
    worktreeRef: null,
    hydrated: true,
    lastEventId: "",
    processedEventKeys: [],
    processedEventKeySet: new Set<string>(),
    completionMessageKeys: [],
    completionMessageKeySet: new Set<string>(),
    ...overrides
  };

  if (!overrides.runs && overrides.executions) {
    runtime.runs = overrides.executions;
  }
  if (!overrides.executions && overrides.runs) {
    runtime.executions = overrides.runs;
  }

  return runtime;
}

function createUserMessage(id: string, queueIndex: number, content: string) {
  return {
    id,
    conversation_id: "conv_1",
    role: "user" as const,
    content,
    queue_index: queueIndex,
    created_at: "2026-02-28T09:00:00Z"
  };
}

function createAssistantMessage(id: string, content: string) {
  return {
    id,
    conversation_id: "conv_1",
    role: "assistant" as const,
    content,
    created_at: "2026-02-28T09:00:10Z"
  };
}

function createExecution(id: string, queueIndex: number, state: "queued" | "pending" | "executing" | "cancelled") {
  return {
    id,
    workspace_id: "ws_1",
    conversation_id: "conv_1",
    message_id: `msg_${queueIndex}`,
    state,
    mode: "default" as const,
    model_id: "gpt-5.3",
    mode_snapshot: "default" as const,
    model_snapshot: {
      model_id: "gpt-5.3"
    },
    project_revision_snapshot: 0,
    queue_index: queueIndex,
    trace_id: `tr_${id}`,
    created_at: "2026-02-28T09:00:00Z",
    updated_at: "2026-02-28T09:00:00Z"
  };
}
