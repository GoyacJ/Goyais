import { computed, ref, watch, type Ref } from "vue";

import type { ExecutionTraceViewModel } from "@/modules/conversation/views/processTrace";

export type ExecutionTraceViewState = ExecutionTraceViewModel & {
  isExpanded: boolean;
};

export function useExecutionTraceState(baseExecutionTraces: Ref<ExecutionTraceViewModel[]>) {
  const traceExpandedByExecutionId = ref<Record<string, boolean>>({});
  const tracePinnedByExecutionId = ref<Record<string, boolean>>({});

  const executionTraces = computed<ExecutionTraceViewState[]>(() =>
    baseExecutionTraces.value.map((trace) => ({
      ...trace,
      isExpanded: traceExpandedByExecutionId.value[trace.executionId] ?? false
    }))
  );
  const activeTraceCount = computed(() => executionTraces.value.filter((trace) => trace.isRunning).length);

  watch(
    baseExecutionTraces,
    (traces) => {
      const expanded = { ...traceExpandedByExecutionId.value };
      const pinned = { ...tracePinnedByExecutionId.value };
      const traceIDs = new Set(traces.map((trace) => trace.executionId));

      for (const executionId of Object.keys(expanded)) {
        if (!traceIDs.has(executionId)) {
          delete expanded[executionId];
          delete pinned[executionId];
        }
      }
      for (const trace of traces) {
        if (!(trace.executionId in expanded)) {
          expanded[trace.executionId] = false;
          continue;
        }
        if (!pinned[trace.executionId] && !trace.isRunning) {
          expanded[trace.executionId] = false;
        }
      }

      traceExpandedByExecutionId.value = expanded;
      tracePinnedByExecutionId.value = pinned;
    },
    { immediate: true }
  );

  function toggleExecutionTrace(executionId: string, expanded: boolean): void {
    traceExpandedByExecutionId.value = {
      ...traceExpandedByExecutionId.value,
      [executionId]: expanded
    };
    tracePinnedByExecutionId.value = {
      ...tracePinnedByExecutionId.value,
      [executionId]: true
    };
  }

  return {
    activeTraceCount,
    executionTraces,
    toggleExecutionTrace
  };
}
