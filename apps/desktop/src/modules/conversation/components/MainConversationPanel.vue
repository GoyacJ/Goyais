<template>
  <section class="panel">
    <div ref="conversationAreaRef" class="conversation-area" @scroll="onConversationScroll">
      <template v-for="message in messages" :key="message.id">
        <div class="message-row" :class="message.role">
          <div class="message-body">
            <div class="bubble-head">
              <AppIcon :name="message.role === 'assistant' ? 'bot' : 'user'" :size="12" />
              <span>{{ message.role === "assistant" ? assistantModelLabel : message.role === "user" ? "You" : "System" }}</span>
            </div>
            <div class="bubble">
              <p>{{ message.content }}</p>
            </div>
            <button
              v-if="message.role === 'user' && message.can_rollback"
              class="rollback"
              type="button"
              @click="$emit('rollback', message.id)"
            >
              回滚到此处
            </button>
          </div>
        </div>

        <div
          v-for="trace in getMessageTraces(message)"
          :key="trace.executionId"
          class="trace-host"
        >
          <ExecutionTraceBlock
            :trace="trace"
            :running-actions="getRunningActions(trace.executionId)"
            @toggle-trace="onTraceToggle"
          />
        </div>
      </template>

      <div v-if="executionHint !== ''" class="message-row assistant">
        <div class="message-body">
          <div class="bubble-head">
            <AppIcon name="bot" :size="12" />
            <span>{{ assistantModelLabel }}</span>
          </div>
          <div class="bubble execution-hint">
            <p>{{ executionHint }}</p>
          </div>
        </div>
      </div>

      <div v-if="hasActiveExecution" class="queue-chip">
        <StatusBadge tone="queued" :label="queueChipLabel" />
      </div>
    </div>

    <div class="composer">
      <textarea
        class="draft"
        :value="draft"
        :placeholder="placeholder"
        @input="onDraftInput"
        @keydown="onDraftKeydown"
      ></textarea>

      <div class="composer-actions">
        <div class="left">
          <div class="plus-wrap">
            <button class="action-btn" type="button" aria-label="更多操作" @click="plusOpen = !plusOpen">
              <AppIcon name="plus" :size="14" />
            </button>
            <div v-if="plusOpen" class="plus-menu">
              <button type="button" @click="plusOpen = false">添加资源</button>
              <button type="button" @click="plusOpen = false">插入模板</button>
            </div>
          </div>

          <select class="select" :value="mode" @change="onModeChange">
            <option value="agent">Agent</option>
            <option value="plan">Plan</option>
          </select>

          <select class="select" :value="modelId" :disabled="!hasModelOptions" @change="onModelChange">
            <option v-if="!hasModelOptions" value="">无可用模型</option>
            <option v-for="option in modelSelectOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </div>

        <div class="right">
          <button class="action-btn" type="button" aria-label="停止执行" :disabled="!hasActiveExecution" @click="$emit('stop')">
            <AppIcon name="square" :size="12" />
          </button>
          <button class="action-btn send" type="button" aria-label="发送消息" @click="$emit('send')">
            <AppIcon name="arrow-up" :size="12" />
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed, nextTick, onMounted, onUpdated, ref } from "vue";
import type { ConversationMessage, ConversationMode } from "@/shared/types/api";
import type { ExecutionTraceStep, ExecutionTraceViewModel } from "@/modules/conversation/views/processTrace";
import type { RunningActionViewModel } from "@/modules/conversation/views/runningActions";
import ExecutionTraceBlock from "@/modules/conversation/components/ExecutionTraceBlock.vue";
import AppIcon from "@/shared/ui/AppIcon.vue";
import StatusBadge from "@/shared/ui/StatusBadge.vue";

type ExecutionTrace = ExecutionTraceViewModel & { isExpanded: boolean; steps: ExecutionTraceStep[] };

const props = withDefaults(
  defineProps<{
    messages: ConversationMessage[];
    queuedCount: number;
    pendingCount: number;
    executingCount: number;
    hasActiveExecution: boolean;
    activeTraceCount: number;
    executionTraces: ExecutionTrace[];
    runningActions: RunningActionViewModel[];
    draft: string;
    mode: ConversationMode;
    modelId: string;
    placeholder: string;
    modelOptions?: Array<{ value: string; label: string }>;
  }>(),
  {
    modelOptions: () => []
  }
);

const modelSelectOptions = computed(() => {
  return (props.modelOptions ?? [])
    .map((item) => ({
      value: item.value.trim(),
      label: item.label.trim() || item.value.trim()
    }))
    .filter((item) => item.value !== "")
    .filter((item, index, source) => source.findIndex((candidate) => candidate.value === item.value) === index);
});

const hasModelOptions = computed(() => modelSelectOptions.value.length > 0);
const assistantModelLabel = computed(() => {
  const normalized = props.modelId.trim();
  if (normalized === "") {
    return "Assistant";
  }
  const selected = modelSelectOptions.value.find((item) => item.value === normalized);
  return selected?.label ?? normalized;
});
const executionHint = computed(() => {
  if (props.activeTraceCount > 0 || props.runningActions.length > 0) {
    return "";
  }
  if (props.pendingCount > 0 || props.executingCount > 0) {
    return "正在思考…";
  }
  return "";
});
const queueChipLabel = computed(() => {
  if (props.queuedCount > 0) {
    return `运行中，队列 ${props.queuedCount}`;
  }
  return "运行中，可停止";
});
const executionTracesByMessageId = computed(() => {
  const mapped = new Map<string, ExecutionTrace[]>();
  for (const trace of props.executionTraces) {
    const messageId = trace.messageId.trim();
    if (messageId === "") {
      continue;
    }
    const list = mapped.get(messageId) ?? [];
    list.push(trace);
    mapped.set(messageId, list);
  }
  return mapped;
});
const executionTracesByQueueIndex = computed(() => {
  const mapped = new Map<number, ExecutionTrace[]>();
  for (const trace of props.executionTraces) {
    const list = mapped.get(trace.queueIndex) ?? [];
    list.push(trace);
    mapped.set(trace.queueIndex, list);
  }
  return mapped;
});
const runningActionsByExecutionId = computed(() => {
  const mapped = new Map<string, RunningActionViewModel[]>();
  for (const action of props.runningActions) {
    const executionID = action.executionId.trim();
    if (executionID === "") {
      continue;
    }
    const list = mapped.get(executionID) ?? [];
    list.push(action);
    mapped.set(executionID, list);
  }
  return mapped;
});

const emit = defineEmits<{
  (event: "send"): void;
  (event: "stop"): void;
  (event: "rollback", messageId: string): void;
  (event: "toggle-trace", executionId: string, expanded: boolean): void;
  (event: "update:draft", value: string): void;
  (event: "update:mode", value: ConversationMode): void;
  (event: "update:model", value: string): void;
}>();

const plusOpen = ref(false);
const conversationAreaRef = ref<HTMLElement | null>(null);
const shouldStickToBottom = ref(true);

function onDraftInput(event: Event): void {
  emit("update:draft", (event.target as HTMLTextAreaElement).value);
}

function onModeChange(event: Event): void {
  emit("update:mode", (event.target as HTMLSelectElement).value as ConversationMode);
}

function onModelChange(event: Event): void {
  emit("update:model", (event.target as HTMLSelectElement).value);
}

function getRunningActions(executionId: string): RunningActionViewModel[] {
  return runningActionsByExecutionId.value.get(executionId) ?? [];
}

function getMessageTraces(message: ConversationMessage): ExecutionTrace[] {
  if (message.role !== "user") {
    return [];
  }
  const directMatches = executionTracesByMessageId.value.get(message.id) ?? [];
  const queueMatches = typeof message.queue_index === "number"
    ? executionTracesByQueueIndex.value.get(message.queue_index) ?? []
    : [];
  if (directMatches.length === 0) {
    return queueMatches;
  }
  if (queueMatches.length === 0) {
    return directMatches;
  }
  const merged = new Map<string, ExecutionTrace>();
  for (const item of directMatches) {
    merged.set(item.executionId, item);
  }
  for (const item of queueMatches) {
    merged.set(item.executionId, item);
  }
  return [...merged.values()];
}

function onTraceToggle(executionId: string, expanded: boolean): void {
  emit("toggle-trace", executionId, expanded);
}

function onDraftKeydown(event: KeyboardEvent): void {
  if (event.key !== "Enter") {
    return;
  }
  if (event.shiftKey || event.ctrlKey || event.altKey || event.metaKey || event.isComposing) {
    return;
  }
  event.preventDefault();
  emit("send");
}

function onConversationScroll(): void {
  const element = conversationAreaRef.value;
  if (!element) {
    return;
  }
  const remaining = element.scrollHeight - element.scrollTop - element.clientHeight;
  shouldStickToBottom.value = remaining <= 32;
}

function scrollConversationToBottom(): void {
  const element = conversationAreaRef.value;
  if (!element) {
    return;
  }
  element.scrollTop = element.scrollHeight;
}

onMounted(() => {
  void nextTick(scrollConversationToBottom);
});

onUpdated(() => {
  if (!shouldStickToBottom.value) {
    return;
  }
  void nextTick(scrollConversationToBottom);
});
</script>

<style scoped src="./MainConversationPanel.css"></style>
