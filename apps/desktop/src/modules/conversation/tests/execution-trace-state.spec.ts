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
    summary: "已思考 3s，已调用 1 个工具",
    steps: [],
    ...overrides
  };
}

describe("execution trace expansion state", () => {
  it("keeps running traces collapsed by default and stays collapsed after finished", async () => {
    const baseTraces = ref<ExecutionTraceViewModel[]>([createTrace()]);
    const { executionTraces } = useExecutionTraceState(baseTraces);

    expect(executionTraces.value[0]?.isExpanded).toBe(false);

    baseTraces.value = [createTrace({ state: "completed", isRunning: false })];
    await nextTick();
    expect(executionTraces.value[0]?.isExpanded).toBe(false);
  });

  it("keeps user preference after manual toggle", async () => {
    const baseTraces = ref<ExecutionTraceViewModel[]>([createTrace()]);
    const { executionTraces, toggleExecutionTrace } = useExecutionTraceState(baseTraces);

    toggleExecutionTrace("exec_trace_state_1", false);
    expect(executionTraces.value[0]?.isExpanded).toBe(false);

    baseTraces.value = [createTrace({ state: "executing", isRunning: true })];
    await nextTick();
    expect(executionTraces.value[0]?.isExpanded).toBe(false);
  });
});
