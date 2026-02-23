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

      <div v-if="queuedCount > 0" class="queue-chip">
        <StatusBadge tone="queued" :label="`队列中 ${queuedCount}`" />
      </div>
    </div>

    <div class="composer">
      <p class="placeholder">{{ placeholder }}</p>

      <div class="composer-actions">
        <div class="left">
          <button class="action-btn" type="button">
            <IconSymbol name="add" :size="14" />
          </button>

          <select class="select" :value="mode" @change="onModeChange">
            <option value="agent">Agent</option>
            <option value="plan">Plan</option>
          </select>

          <select class="select" :value="modelId" @change="onModelChange">
            <option value="gpt-4.1">gpt-4.1</option>
            <option value="gpt-4.1-mini">gpt-4.1-mini</option>
            <option value="claude-sonnet-4.5">claude-sonnet-4.5</option>
          </select>
        </div>

        <div class="right">
          <button
            class="action-btn send"
            type="button"
            @click="$emit('send')"
          >
            <IconSymbol name="send" :size="13" />
          </button>
          <button
            class="action-btn"
            type="button"
            :disabled="!hasActiveExecution"
            @click="$emit('stop')"
          >
            <IconSymbol name="stop" :size="13" />
          </button>
        </div>
      </div>

      <textarea
        class="draft"
        :value="draft"
        @input="onDraftInput"
      ></textarea>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { ConversationMessage, ConversationMode } from "@/shared/types/api";
import IconSymbol from "@/shared/ui/IconSymbol.vue";
import StatusBadge from "@/shared/ui/StatusBadge.vue";

defineProps<{
  messages: ConversationMessage[];
  queuedCount: number;
  hasActiveExecution: boolean;
  draft: string;
  mode: ConversationMode;
  modelId: string;
  placeholder: string;
}>();

const emit = defineEmits<{
  (event: "send"): void;
  (event: "stop"): void;
  (event: "rollback", messageId: string): void;
  (event: "update:draft", value: string): void;
  (event: "update:mode", value: ConversationMode): void;
  (event: "update:model", value: string): void;
}>();

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
  padding: var(--global-space-12);
  display: grid;
  gap: var(--global-space-8);
}
.placeholder {
  color: var(--semantic-text);
  font-size: var(--global-font-size-13);
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
.draft {
  min-height: 56px;
  border: 0;
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  color: var(--semantic-text);
  resize: vertical;
  font: inherit;
}
</style>
