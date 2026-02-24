<template>
  <BaseModal :open="open">
    <template #title>
      <h3 class="title">{{ title }}</h3>
    </template>

    <div class="body">
      <textarea v-model="draftText" class="editor" spellcheck="false" />
      <p v-if="errorMessage !== ''" class="error">{{ errorMessage }}</p>
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="emit('close')">关闭</BaseButton>
      <BaseButton variant="primary" @click="saveDraft">保存</BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";

import BaseModal from "@/shared/ui/BaseModal.vue";
import BaseButton from "@/shared/ui/BaseButton.vue";

const props = withDefaults(
  defineProps<{
    open: boolean;
    payload: Record<string, unknown> | null;
    title?: string;
  }>(),
  {
    title: "MCP 配置"
  }
);

const emit = defineEmits<{
  (event: "close"): void;
  (event: "save", payload: Record<string, unknown>): void;
}>();

const draftText = ref("");
const errorMessage = ref("");

watch(
  () => [props.open, props.payload] as const,
  ([open]) => {
    if (!open) return;
    draftText.value = JSON.stringify(props.payload ?? {}, null, 2);
    errorMessage.value = "";
  },
  { immediate: true }
);

function saveDraft(): void {
  try {
    const parsed = JSON.parse(draftText.value) as Record<string, unknown>;
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
      errorMessage.value = "MCP 配置必须是 JSON 对象";
      return;
    }
    emit("save", parsed);
    emit("close");
  } catch {
    errorMessage.value = "JSON 格式错误，请检查后重试";
  }
}
</script>

<style scoped>
.title {
  margin: 0;
}

.body {
  display: grid;
  gap: var(--global-space-8);
}

.editor {
  min-height: min(60vh, 520px);
  max-height: min(60vh, 520px);
  width: 100%;
  resize: vertical;
  border: 1px solid var(--semantic-border);
  border-radius: var(--component-input-radius);
  background: var(--component-input-bg);
  padding: var(--global-space-8);
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
  font-family: var(--global-font-family-ui);
  white-space: pre-wrap;
  word-break: break-word;
}

.error {
  margin: 0;
  color: var(--semantic-danger);
  font-size: var(--global-font-size-12);
}
</style>
