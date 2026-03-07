<template>
  <div v-if="visible" class="commit-dialog-overlay" role="presentation" @click="onBackdropClick">
    <section class="commit-dialog" role="dialog" aria-modal="true" :aria-label="t('session.commitDialog.title')" @click.stop>
      <header class="commit-dialog-head">
        <h3>{{ t("session.commitDialog.title") }}</h3>
      </header>
      <label class="commit-dialog-label" for="commit-message">{{ t("session.commitDialog.messageLabel") }}</label>
      <textarea
        id="commit-message"
        ref="textareaRef"
        class="commit-dialog-textarea"
        :value="message"
        @input="onInput"
      ></textarea>
      <div class="commit-dialog-actions">
        <button type="button" class="commit-dialog-btn" @click="$emit('close')">
          {{ t("session.commitDialog.cancel") }}
        </button>
        <button
          type="button"
          class="commit-dialog-btn primary"
          :disabled="pending || message.trim() === ''"
          @click="onConfirm"
        >
          {{ pending ? t("session.commitDialog.confirming") : t("session.commitDialog.confirm") }}
        </button>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from "vue";

import { useI18n } from "@/shared/i18n";

const props = withDefaults(defineProps<{
  visible: boolean;
  defaultMessage: string;
  pending?: boolean;
}>(), {
  pending: false
});

const emit = defineEmits<{
  (event: "close"): void;
  (event: "confirm", message: string): void;
}>();

const { t } = useI18n();
const message = ref("");
const textareaRef = ref<HTMLTextAreaElement | null>(null);

watch(
  () => [props.visible, props.defaultMessage] as const,
  async ([visible]) => {
    if (!visible) {
      return;
    }
    message.value = props.defaultMessage;
    await nextTick();
    textareaRef.value?.focus();
    textareaRef.value?.select();
  },
  { immediate: true }
);

function onInput(event: Event): void {
  message.value = (event.target as HTMLTextAreaElement).value;
}

function onConfirm(): void {
  emit("confirm", message.value.trim());
}

function onBackdropClick(): void {
  if (!props.pending) {
    emit("close");
  }
}
</script>

<style scoped>
.commit-dialog-overlay {
  position: fixed;
  inset: 0;
  display: grid;
  place-items: center;
  background: rgba(0, 0, 0, 0.45);
  z-index: 30;
}

.commit-dialog {
  width: min(560px, calc(100vw - 24px));
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  border-radius: 12px;
  background: var(--panel-bg, #fff);
  border: 1px solid var(--line, rgba(0, 0, 0, 0.12));
  box-shadow: 0 18px 48px rgba(0, 0, 0, 0.24);
}

.commit-dialog-head h3 {
  margin: 0;
  font-size: 15px;
}

.commit-dialog-label {
  font-size: 12px;
  color: var(--muted, #666);
}

.commit-dialog-textarea {
  width: 100%;
  min-height: 96px;
  resize: vertical;
  padding: 10px;
  border-radius: 8px;
  border: 1px solid var(--line, rgba(0, 0, 0, 0.12));
  background: transparent;
  color: inherit;
  font: inherit;
}

.commit-dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.commit-dialog-btn {
  border: 1px solid var(--line, rgba(0, 0, 0, 0.2));
  border-radius: 8px;
  padding: 6px 10px;
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.commit-dialog-btn.primary {
  background: var(--accent, #2b72ff);
  border-color: var(--accent, #2b72ff);
  color: #fff;
}

.commit-dialog-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
</style>
