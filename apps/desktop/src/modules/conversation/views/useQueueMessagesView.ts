import { computed, type Ref } from "vue";

import type { ConversationRuntime } from "@/modules/conversation/store/state";
import type { ConversationMessage, Execution } from "@/shared/types/api";

export type QueuedMessageViewModel = {
  executionId: string;
  messageId: string;
  queueIndex: number;
  content: string;
  preview: string;
};

type UserMessageLookup = {
  byID: Map<string, ConversationMessage>;
  byQueueIndex: Map<number, ConversationMessage>;
};

export function useQueueMessagesView(runtime: Ref<ConversationRuntime | undefined>) {
  const userMessages = computed<UserMessageLookup>(() => {
    const byID = new Map<string, ConversationMessage>();
    const byQueueIndex = new Map<number, ConversationMessage>();
    for (const message of runtime.value?.messages ?? []) {
      if (message.role !== "user") {
        continue;
      }
      const messageID = message.id.trim();
      if (messageID !== "") {
        byID.set(messageID, message);
      }
      if (typeof message.queue_index === "number") {
        byQueueIndex.set(message.queue_index, message);
      }
    }
    return { byID, byQueueIndex };
  });

  const queuedExecutions = computed(() =>
    [...(runtime.value?.executions ?? [])]
      .filter((execution) => execution.state === "queued")
      .sort((left, right) => {
        if (left.queue_index !== right.queue_index) {
          return left.queue_index - right.queue_index;
        }
        const createdComparison = left.created_at.localeCompare(right.created_at);
        if (createdComparison !== 0) {
          return createdComparison;
        }
        return left.id.localeCompare(right.id);
      })
  );

  const removedQueuedExecutionIDs = computed(() => {
    const executionIDs = new Set<string>();
    for (const event of runtime.value?.events ?? []) {
      if (event.type !== "execution_stopped") {
        continue;
      }
      if (event.payload.action !== "stop" || event.payload.source !== "run_control") {
        continue;
      }
      const executionID = event.execution_id.trim();
      if (executionID !== "") {
        executionIDs.add(executionID);
      }
    }
    return executionIDs;
  });

  const hiddenQueueIndexes = computed(() => {
    const indexes = new Set<number>();
    for (const execution of runtime.value?.executions ?? []) {
      if (isHiddenExecution(execution, removedQueuedExecutionIDs.value)) {
        indexes.add(execution.queue_index);
      }
    }
    return indexes;
  });

  const queuedMessages = computed<QueuedMessageViewModel[]>(() =>
    queuedExecutions.value.map((execution) => {
      const message = resolveUserMessageForExecution(execution, userMessages.value);
      const content = message?.content ?? "";
      return {
        executionId: execution.id,
        messageId: (message?.id ?? execution.message_id).trim(),
        queueIndex: execution.queue_index,
        content,
        preview: buildMessagePreview(content)
      };
    })
  );

  const visibleMessages = computed<ConversationMessage[]>(() => {
    const hiddenIndexes = hiddenQueueIndexes.value;
    return (runtime.value?.messages ?? []).filter((message) => {
      if (message.role !== "user") {
        return true;
      }
      if (typeof message.queue_index !== "number") {
        return true;
      }
      return !hiddenIndexes.has(message.queue_index);
    });
  });

  const visibleTraceExecutionIds = computed(() => {
    const executionIDs = new Set<string>();
    for (const execution of runtime.value?.executions ?? []) {
      if (isHiddenExecution(execution, removedQueuedExecutionIDs.value)) {
        continue;
      }
      executionIDs.add(execution.id);
    }
    return executionIDs;
  });

  return {
    queuedMessages,
    visibleMessages,
    visibleTraceExecutionIds
  };
}

function resolveUserMessageForExecution(execution: Execution, userMessages: UserMessageLookup): ConversationMessage | undefined {
  if (typeof execution.queue_index === "number") {
    const byQueueIndex = userMessages.byQueueIndex.get(execution.queue_index);
    if (byQueueIndex) {
      return byQueueIndex;
    }
  }
  const messageID = execution.message_id.trim();
  if (messageID !== "") {
    return userMessages.byID.get(messageID);
  }
  return undefined;
}

function isHiddenExecution(execution: Execution, removedQueuedExecutionIDs: Set<string>): boolean {
  if (execution.state === "queued") {
    return true;
  }
  if (execution.state === "cancelled" && removedQueuedExecutionIDs.has(execution.id)) {
    return true;
  }
  return false;
}

function buildMessagePreview(content: string): string {
  const normalized = content.trim().replace(/\s+/g, " ");
  if (normalized.length <= 64) {
    return normalized;
  }
  return `${normalized.slice(0, 61)}...`;
}
