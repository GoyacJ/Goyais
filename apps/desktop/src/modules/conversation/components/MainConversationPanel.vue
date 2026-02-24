<template>
  <section class="panel">
    <div ref="conversationAreaRef" class="conversation-area" @scroll="onConversationScroll">
      <div
        v-for="message in messages"
        :key="message.id"
        class="message-row"
        :class="message.role"
      >
        <div class="message-body">
          <div class="bubble-head">
            <AppIcon :name="message.role === 'assistant' ? 'bot' : 'user'" :size="12" />
            <span>{{ message.role === "assistant" ? modelId : message.role === "user" ? "You" : "System" }}</span>
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

      <div v-if="hasActiveExecution" class="queue-chip">
        <StatusBadge tone="queued" :label="queuedCount > 0 ? `运行中，队列 ${queuedCount}` : '运行中，可停止'" />
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
            <button class="action-btn" type="button" @click="plusOpen = !plusOpen">
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
          <button class="action-btn" type="button" :disabled="!hasActiveExecution" @click="$emit('stop')">
            <AppIcon name="square" :size="12" />
          </button>
          <button class="action-btn send" type="button" @click="$emit('send')">
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
import AppIcon from "@/shared/ui/AppIcon.vue";
import StatusBadge from "@/shared/ui/StatusBadge.vue";

const props = withDefaults(
  defineProps<{
    messages: ConversationMessage[];
    queuedCount: number;
    hasActiveExecution: boolean;
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

const emit = defineEmits<{
  (event: "send"): void;
  (event: "stop"): void;
  (event: "rollback", messageId: string): void;
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

<style scoped>
.panel {
  display: grid;
  grid-template-rows: 1fr auto;
  gap: var(--global-space-8);
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.conversation-area {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-12);
  background: var(--semantic-bg);
  padding: var(--global-space-16) var(--global-space-12);
  display: grid;
  gap: var(--global-space-10);
  align-content: start;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
}

.message-row {
  display: flex;
  width: 100%;
  justify-content: flex-start;
  padding: 0 var(--global-space-2);
}

.message-row.user {
  justify-content: flex-end;
}

.message-row.system {
  justify-content: center;
}

.message-body {
  width: fit-content;
  max-width: min(760px, 86%);
  display: grid;
  gap: var(--global-space-4);
}

.bubble {
  width: fit-content;
  max-width: 100%;
  padding: var(--global-space-8) var(--global-space-12);
  background: var(--component-shell-content-bg-elevated);
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-12) var(--global-radius-12) var(--global-radius-12) var(--global-radius-8);
  color: var(--semantic-text);
  box-shadow: var(--global-shadow-1);
}

.message-row.user .message-body {
  justify-items: end;
}

.message-row.user .bubble {
  background: var(--component-sidebar-item-bg-active);
  border-color: var(--component-shell-divider);
  border-radius: var(--global-radius-12) var(--global-radius-12) var(--global-radius-8) var(--global-radius-12);
  text-align: left;
}

.message-row.system .message-body {
  width: min(680px, 92%);
  justify-items: center;
}

.message-row.system .bubble {
  background: transparent;
  border-style: dashed;
  text-align: center;
}

.bubble-head {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-4);
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
  line-height: 1.3;
  padding: 0 var(--global-space-2);
}

.message-row.user .bubble-head {
  color: var(--semantic-text-muted);
}

p {
  margin: 0;
  white-space: pre-wrap;
  line-height: 1.6;
  font-size: var(--global-font-size-13);
}

.rollback {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
  padding: var(--global-space-4) var(--global-space-8);
  justify-self: end;
  opacity: 0;
  pointer-events: none;
  transform: translateY(-2px);
  transition: opacity 0.16s ease, transform 0.16s ease;
}

.message-row.user:hover .rollback,
.message-row.user:focus-within .rollback {
  opacity: 1;
  pointer-events: auto;
  transform: translateY(0);
}

.queue-chip {
  justify-self: center;
}

.composer {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-12);
  background: var(--semantic-bg);
  padding: var(--global-space-8);
  display: grid;
  gap: var(--global-space-8);
}

.draft {
  min-height: 76px;
  border: 0;
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  color: var(--semantic-text);
  resize: vertical;
  font: inherit;
  padding: var(--global-space-8);
}

.composer-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.left,
.right {
  display: inline-flex;
  gap: var(--global-space-8);
  align-items: center;
}

.plus-wrap {
  position: relative;
}

.plus-menu {
  position: absolute;
  left: 0;
  bottom: calc(100% + 6px);
  width: 140px;
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface);
  border: 1px solid var(--semantic-border);
  display: grid;
  gap: 1px;
  padding: var(--global-space-4);
}

.plus-menu button {
  border: 0;
  border-radius: var(--global-radius-8);
  background: transparent;
  color: var(--semantic-text);
  padding: var(--global-space-4) var(--global-space-8);
  text-align: left;
  font-size: var(--global-font-size-11);
}

.action-btn {
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: 999px;
  background: var(--semantic-surface);
  color: var(--semantic-text);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.action-btn.send {
  background: var(--semantic-primary);
  color: var(--semantic-bg);
}

.select {
  border: 0;
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface);
  color: var(--semantic-text);
  height: 28px;
  padding: 0 var(--global-space-8);
  font-size: var(--global-font-size-12);
}

@media (max-width: 960px) {
  .conversation-area {
    padding: var(--global-space-10) var(--global-space-8);
    gap: var(--global-space-8);
  }

  .message-body {
    max-width: 94%;
  }
}
</style>
