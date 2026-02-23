<template>
  <section class="panel">
    <div class="conversation-area">
      <div
        v-for="message in messages"
        :key="message.id"
        class="message-row"
        :class="message.role"
      >
        <div class="bubble">
          <div class="bubble-head">
            <AppIcon :name="message.role === 'assistant' ? 'bot' : 'user'" :size="12" />
            <span>{{ message.role === "assistant" ? "AI" : message.role === "user" ? "You" : "System" }}</span>
          </div>
          <p>{{ message.content }}</p>
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

          <select class="select" :value="modelId" @change="onModelChange">
            <option v-for="option in modelOptions" :key="option" :value="option">{{ option }}</option>
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
import { ref } from "vue";

import type { ConversationMessage, ConversationMode } from "@/shared/types/api";
import AppIcon from "@/shared/ui/AppIcon.vue";
import StatusBadge from "@/shared/ui/StatusBadge.vue";

withDefaults(
  defineProps<{
    messages: ConversationMessage[];
    queuedCount: number;
    hasActiveExecution: boolean;
    draft: string;
    mode: ConversationMode;
    modelId: string;
    placeholder: string;
    modelOptions?: string[];
  }>(),
  {
    modelOptions: () => ["gpt-4.1", "gpt-4.1-mini", "gemini-2.0-flash", "qwen-max"]
  }
);

const emit = defineEmits<{
  (event: "send"): void;
  (event: "stop"): void;
  (event: "rollback", messageId: string): void;
  (event: "update:draft", value: string): void;
  (event: "update:mode", value: ConversationMode): void;
  (event: "update:model", value: string): void;
}>();

const plusOpen = ref(false);

function onDraftInput(event: Event): void {
  emit("update:draft", (event.target as HTMLTextAreaElement).value);
}

function onModeChange(event: Event): void {
  emit("update:mode", (event.target as HTMLSelectElement).value as ConversationMode);
}

function onModelChange(event: Event): void {
  emit("update:model", (event.target as HTMLSelectElement).value);
}
</script>

<style scoped>
.panel {
  display: grid;
  grid-template-rows: 1fr auto;
  gap: var(--global-space-8);
  min-height: 0;
}

.conversation-area {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-12);
  background: var(--semantic-bg);
  padding: var(--global-space-8);
  display: grid;
  gap: var(--global-space-8);
  align-content: start;
  overflow: auto;
}

.message-row {
  display: flex;
}

.message-row.user {
  justify-content: flex-end;
}

.bubble {
  width: min(760px, 88%);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-12);
  background: var(--semantic-surface);
  color: var(--semantic-text-muted);
  display: grid;
  gap: var(--global-space-8);
}

.message-row.user .bubble {
  background: var(--semantic-surface-2);
  color: var(--semantic-text);
}

.bubble-head {
  display: inline-flex;
  align-items: center;
  gap: var(--global-space-4);
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
}

p {
  margin: 0;
  white-space: pre-wrap;
}

.rollback {
  border: 0;
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface);
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
  padding: var(--global-space-4) var(--global-space-8);
  justify-self: end;
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
</style>
