import { nextTick, ref } from "vue";
import { describe, expect, it } from "vitest";

import { useRunTraceState } from "@/modules/session/views/useRunTraceState";
import type { RunTraceViewModel } from "@/modules/session/views/processTrace";
import type { SessionMessage } from "@/shared/types/api";

function createTrace(overrides?: Partial<RunTraceViewModel>): RunTraceViewModel {
  return {
    executionId: "exec_trace_state_1",
    messageId: "msg_trace_state_1",
    queueIndex: 0,
    state: "executing",
    isRunning: true,
    summaryPrimary: "已思考 3s · 调用 1 个工具",
    summarySecondary: "消息执行 3s",
    summaryTone: "primary",
    steps: [],
    ...overrides
  };
}

describe("run trace selection state", () => {
  it("selects latest trace by default", () => {
    const baseTraces = ref<RunTraceViewModel[]>([
      createTrace({ executionId: "exec_trace_state_1" }),
      createTrace({ executionId: "exec_trace_state_2", queueIndex: 1 })
    ]);
    const baseMessages = ref<SessionMessage[]>([
      createUserMessage("msg_trace_state_1", 0),
      createUserMessage("msg_trace_state_2", 1)
    ]);
    const { selectedRunTrace, selectedTraceExecutionId, selectedTraceMessageId } = useRunTraceState(baseTraces, baseMessages);

    expect(selectedTraceMessageId.value).toBe("msg_trace_state_2");
    expect(selectedTraceExecutionId.value).toBe("exec_trace_state_2");
    expect(selectedRunTrace.value?.executionId).toBe("exec_trace_state_2");
  });

  it("keeps user-selected trace when list updates", async () => {
    const baseTraces = ref<RunTraceViewModel[]>([
      createTrace({ executionId: "exec_trace_state_1" }),
      createTrace({ executionId: "exec_trace_state_2", queueIndex: 1 })
    ]);
    const baseMessages = ref<SessionMessage[]>([
      createUserMessage("msg_trace_state_1", 0),
      createUserMessage("msg_trace_state_2", 1)
    ]);
    const { selectedRunTrace, selectedTraceExecutionId, selectRunTrace } = useRunTraceState(baseTraces, baseMessages);

    selectRunTrace("exec_trace_state_1");
    expect(selectedTraceExecutionId.value).toBe("exec_trace_state_1");

    baseTraces.value = [
      createTrace({ executionId: "exec_trace_state_1", state: "completed", isRunning: false }),
      createTrace({ executionId: "exec_trace_state_2", queueIndex: 1 })
    ];
    await nextTick();

    expect(selectedTraceExecutionId.value).toBe("exec_trace_state_1");
    expect(selectedRunTrace.value?.executionId).toBe("exec_trace_state_1");
  });

  it("falls back to latest trace when selected trace disappears", async () => {
    const baseTraces = ref<RunTraceViewModel[]>([
      createTrace({ executionId: "exec_trace_state_1" }),
      createTrace({ executionId: "exec_trace_state_2", queueIndex: 1 })
    ]);
    const baseMessages = ref<SessionMessage[]>([
      createUserMessage("msg_trace_state_1", 0),
      createUserMessage("msg_trace_state_2", 1)
    ]);
    const { selectedTraceExecutionId, selectRunTrace } = useRunTraceState(baseTraces, baseMessages);

    selectRunTrace("exec_trace_state_1");
    expect(selectedTraceExecutionId.value).toBe("exec_trace_state_1");

    baseTraces.value = [createTrace({ executionId: "exec_trace_state_2", queueIndex: 1 })];
    await nextTick();

    expect(selectedTraceExecutionId.value).toBe("exec_trace_state_2");
  });
});

function createUserMessage(id: string, queueIndex: number): SessionMessage {
  return {
    id,
    session_id: "conv_trace_state_1",
    role: "user",
    content: "hello",
    queue_index: queueIndex,
    created_at: "2026-02-24T00:00:00Z"
  };
}
