import { computed, ref, watch, type Ref } from "vue";

import type { ExecutionTraceViewModel } from "@/modules/conversation/views/processTrace";
import type { SessionMessage } from "@/shared/types/api";

type TraceMessageItem = {
  id: string;
  traces: ExecutionTraceViewModel[];
};

export function useExecutionTraceState(
  baseExecutionTraces: Ref<ExecutionTraceViewModel[]>,
  baseMessages: Ref<SessionMessage[]>
) {
  const selectedTraceMessageId = ref("");
  const selectedTraceExecutionId = ref("");
  const executionTraces = computed<ExecutionTraceViewModel[]>(() => baseExecutionTraces.value);
  const activeTraceCount = computed(() => executionTraces.value.filter((trace) => trace.isRunning).length);
  const traceMessageItems = computed<TraceMessageItem[]>(() => {
    const tracesByMessageId = new Map<string, ExecutionTraceViewModel[]>();
    const tracesByQueueIndex = new Map<number, ExecutionTraceViewModel[]>();
    for (const trace of executionTraces.value) {
      const messageID = trace.messageId.trim();
      if (messageID !== "") {
        const byMessageId = tracesByMessageId.get(messageID) ?? [];
        byMessageId.push(trace);
        tracesByMessageId.set(messageID, byMessageId);
      }
      const byQueueIndex = tracesByQueueIndex.get(trace.queueIndex) ?? [];
      byQueueIndex.push(trace);
      tracesByQueueIndex.set(trace.queueIndex, byQueueIndex);
    }

    const result: TraceMessageItem[] = [];
    for (const message of baseMessages.value) {
      if (message.role !== "user") {
        continue;
      }
      const messageID = message.id.trim();
      if (messageID === "") {
        continue;
      }
      const directMatches = tracesByMessageId.get(messageID) ?? [];
      const queueMatches = typeof message.queue_index === "number"
        ? tracesByQueueIndex.get(message.queue_index) ?? []
        : [];
      const merged = new Map<string, ExecutionTraceViewModel>();
      for (const trace of directMatches) {
        merged.set(trace.executionId, trace);
      }
      for (const trace of queueMatches) {
        merged.set(trace.executionId, trace);
      }
      result.push({
        id: messageID,
        traces: [...merged.values()].sort((left, right) => left.queueIndex - right.queueIndex)
      });
    }
    return result;
  });
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
    traceMessageItems,
    (messageItems) => {
      if (messageItems.length <= 0) {
        selectedTraceMessageId.value = "";
        selectedTraceExecutionId.value = "";
        return;
      }
      const selectedMessageID = selectedTraceMessageId.value.trim();
      let messageItem = messageItems.find((item) => item.id === selectedMessageID);
      if (!messageItem) {
        messageItem = [...messageItems].reverse().find((item) => item.traces.length > 0) ?? messageItems[messageItems.length - 1];
      }
      selectedTraceMessageId.value = messageItem?.id ?? "";
      const traces = messageItem?.traces ?? [];
      if (traces.length <= 0) {
        selectedTraceExecutionId.value = "";
        return;
      }
      const selectedExecutionId = selectedTraceExecutionId.value.trim();
      if (selectedExecutionId !== "" && traces.some((trace) => trace.executionId === selectedExecutionId)) {
        return;
      }
      selectedTraceExecutionId.value = traces[traces.length - 1]?.executionId ?? "";
    },
    { immediate: true }
  );

  function selectTraceMessage(messageId: string): void {
    const normalizedMessageID = messageId.trim();
    if (normalizedMessageID === "") {
      return;
    }
    const matchedMessage = traceMessageItems.value.find((item) => item.id === normalizedMessageID);
    if (!matchedMessage) {
      return;
    }
    selectedTraceMessageId.value = normalizedMessageID;
    const selectedExecutionId = selectedTraceExecutionId.value.trim();
    if (selectedExecutionId !== "" && matchedMessage.traces.some((trace) => trace.executionId === selectedExecutionId)) {
      return;
    }
    selectedTraceExecutionId.value = matchedMessage.traces[matchedMessage.traces.length - 1]?.executionId ?? "";
  }

  function selectExecutionTrace(executionId: string): void {
    const normalizedExecutionId = executionId.trim();
    if (normalizedExecutionId === "") {
      return;
    }
    const matchedTrace = executionTraces.value.find((trace) => trace.executionId === normalizedExecutionId);
    if (!matchedTrace) {
      return;
    }
    selectedTraceExecutionId.value = normalizedExecutionId;
    const owningMessage = traceMessageItems.value.find((item) => item.traces.some((trace) => trace.executionId === normalizedExecutionId));
    if (owningMessage) {
      selectedTraceMessageId.value = owningMessage.id;
      return;
    }
    const fallbackMessageID = matchedTrace.messageId.trim();
    if (fallbackMessageID !== "") {
      selectedTraceMessageId.value = fallbackMessageID;
    }
  }

  return {
    activeTraceCount,
    executionTraces,
    selectedExecutionTrace,
    selectedTraceMessageId,
    selectedTraceExecutionId,
    selectTraceMessage,
    selectExecutionTrace
  };
}
