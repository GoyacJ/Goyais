import { nextTick, ref } from "vue";
import { describe, expect, it } from "vitest";

import { useExecutionTraceState } from "@/modules/conversation/views/useExecutionTraceState";
import type { ExecutionTraceViewModel } from "@/modules/conversation/views/processTrace";

function createTrace(overrides?: Partial<ExecutionTraceViewModel>): ExecutionTraceViewModel {
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

describe("execution trace selection state", () => {
  it("selects latest trace by default", () => {
    const baseTraces = ref<ExecutionTraceViewModel[]>([
      createTrace({ executionId: "exec_trace_state_1" }),
      createTrace({ executionId: "exec_trace_state_2", queueIndex: 1 })
    ]);
    const { selectedExecutionTrace, selectedTraceExecutionId } = useExecutionTraceState(baseTraces);

    expect(selectedTraceExecutionId.value).toBe("exec_trace_state_2");
    expect(selectedExecutionTrace.value?.executionId).toBe("exec_trace_state_2");
  });

  it("keeps user-selected trace when list updates", async () => {
    const baseTraces = ref<ExecutionTraceViewModel[]>([
      createTrace({ executionId: "exec_trace_state_1" }),
      createTrace({ executionId: "exec_trace_state_2", queueIndex: 1 })
    ]);
    const { selectedExecutionTrace, selectedTraceExecutionId, selectExecutionTrace } = useExecutionTraceState(baseTraces);

    selectExecutionTrace("exec_trace_state_1");
    expect(selectedTraceExecutionId.value).toBe("exec_trace_state_1");

    baseTraces.value = [
      createTrace({ executionId: "exec_trace_state_1", state: "completed", isRunning: false }),
      createTrace({ executionId: "exec_trace_state_2", queueIndex: 1 })
    ];
    await nextTick();

    expect(selectedTraceExecutionId.value).toBe("exec_trace_state_1");
    expect(selectedExecutionTrace.value?.executionId).toBe("exec_trace_state_1");
  });

  it("falls back to latest trace when selected trace disappears", async () => {
    const baseTraces = ref<ExecutionTraceViewModel[]>([
      createTrace({ executionId: "exec_trace_state_1" }),
      createTrace({ executionId: "exec_trace_state_2", queueIndex: 1 })
    ]);
    const { selectedTraceExecutionId, selectExecutionTrace } = useExecutionTraceState(baseTraces);

    selectExecutionTrace("exec_trace_state_1");
    expect(selectedTraceExecutionId.value).toBe("exec_trace_state_1");

    baseTraces.value = [createTrace({ executionId: "exec_trace_state_2", queueIndex: 1 })];
    await nextTick();

    expect(selectedTraceExecutionId.value).toBe("exec_trace_state_2");
  });
});
