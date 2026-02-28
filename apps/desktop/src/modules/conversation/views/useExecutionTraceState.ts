import { computed, ref, watch, type Ref } from "vue";

import type { ExecutionTraceViewModel } from "@/modules/conversation/views/processTrace";

export function useExecutionTraceState(baseExecutionTraces: Ref<ExecutionTraceViewModel[]>) {
  const selectedTraceExecutionId = ref("");
  const executionTraces = computed<ExecutionTraceViewModel[]>(() => baseExecutionTraces.value);
  const activeTraceCount = computed(() => executionTraces.value.filter((trace) => trace.isRunning).length);
  const selectedExecutionTrace = computed<ExecutionTraceViewModel | undefined>(() => {
    const traces = executionTraces.value;
    if (traces.length <= 0) {
      return undefined;
    }
    const normalizedSelectedExecutionId = selectedTraceExecutionId.value.trim();
    if (normalizedSelectedExecutionId !== "") {
      const selectedTrace = traces.find((trace) => trace.executionId === normalizedSelectedExecutionId);
      if (selectedTrace) {
        return selectedTrace;
      }
    }
    return traces[traces.length - 1];
  });

  watch(
    baseExecutionTraces,
    (traces) => {
      if (traces.length <= 0) {
        selectedTraceExecutionId.value = "";
        return;
      }
      const selectedExecutionId = selectedTraceExecutionId.value.trim();
      if (selectedExecutionId === "") {
        selectedTraceExecutionId.value = traces[traces.length - 1]?.executionId ?? "";
        return;
      }
      const isStillVisible = traces.some((trace) => trace.executionId === selectedExecutionId);
      if (isStillVisible) {
        return;
      }
      selectedTraceExecutionId.value = traces[traces.length - 1]?.executionId ?? "";
    },
    { immediate: true }
  );

  function selectExecutionTrace(executionId: string): void {
    const normalizedExecutionId = executionId.trim();
    if (normalizedExecutionId === "") {
      return;
    }
    const matchedTrace = executionTraces.value.some((trace) => trace.executionId === normalizedExecutionId);
    if (!matchedTrace) {
      return;
    }
    selectedTraceExecutionId.value = normalizedExecutionId;
  }

  return {
    activeTraceCount,
    executionTraces,
    selectedExecutionTrace,
    selectedTraceExecutionId,
    selectExecutionTrace
  };
}
