<template>
  <BaseModal :open="open" @close="emit('close')">
    <template #title>
      <h3 class="title m-0">{{ title }}</h3>
    </template>

    <div class="body grid gap-[var(--global-space-8)]">
      <textarea
        v-model="draftText"
        class="editor min-h-[min(60vh,520px)] max-h-[min(60vh,520px)] w-full resize-y border border-[var(--semantic-border)] rounded-[var(--component-input-radius)] bg-[var(--component-input-bg)] p-[var(--global-space-8)] text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)] [font-family:var(--global-font-family-ui)] whitespace-pre-wrap break-words"
        spellcheck="false"
      />
      <p v-if="errorMessage !== ''" class="error m-0 text-[var(--global-font-size-12)] text-[var(--semantic-danger)]">{{ errorMessage }}</p>
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
