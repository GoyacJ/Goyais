<template>
  <div class="editor-wrap grid grid-cols-2 gap-[var(--global-space-10)] max-[960px]:grid-cols-1">
    <div class="pane grid gap-[var(--global-space-6)]">
      <p v-if="label !== ''" class="pane-title m-0 text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">{{ label }}</p>
      <textarea
        class="editor min-h-[220px] resize-y border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-10)] text-[var(--global-font-size-13)] text-[var(--semantic-text)] [font-family:var(--global-font-family-code)]"
        :placeholder="placeholder"
        :value="modelValue"
        :disabled="disabled"
        @input="onInput"
      />
    </div>

    <div class="pane grid gap-[var(--global-space-6)]">
      <p class="pane-title m-0 text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">实时预览</p>
      <article
        class="preview min-h-[220px] overflow-auto border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] p-[var(--global-space-10)] text-[var(--semantic-text)] [&_h1]:mb-[var(--global-space-8)] [&_h1]:mt-0 [&_h2]:mb-[var(--global-space-8)] [&_h2]:mt-0 [&_h3]:mb-[var(--global-space-8)] [&_h3]:mt-0 [&_h4]:mb-[var(--global-space-8)] [&_h4]:mt-0 [&_h5]:mb-[var(--global-space-8)] [&_h5]:mt-0 [&_h6]:mb-[var(--global-space-8)] [&_h6]:mt-0 [&_p]:mb-[var(--global-space-8)] [&_p]:mt-0 [&_ul]:mb-[var(--global-space-8)] [&_ul]:mt-0 [&_ul]:pl-[var(--global-space-20)] [&_ol]:mb-[var(--global-space-8)] [&_ol]:mt-0 [&_ol]:pl-[var(--global-space-20)] [&_code]:rounded-[var(--global-radius-4)] [&_code]:bg-[var(--semantic-surface-2)] [&_code]:px-[var(--global-space-4)] [&_code]:[font-family:var(--global-font-family-code)] [&_a]:text-[var(--semantic-link)]"
        v-html="renderedHtml"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    modelValue: string;
    label?: string;
    placeholder?: string;
    disabled?: boolean;
  }>(),
  {
    label: "Markdown",
    placeholder: "请输入 Markdown 内容",
    disabled: false
  }
);

const emit = defineEmits<{
  (event: "update:modelValue", value: string): void;
}>();

const renderedHtml = computed(() => renderMarkdown(props.modelValue));

function onInput(event: Event): void {
  emit("update:modelValue", (event.target as HTMLTextAreaElement).value);
}

function renderMarkdown(source: string): string {
  const escaped = escapeHtml(source);
  const lines = escaped.split(/\r?\n/);
  let html = "";
  let listMode: "ul" | "ol" | null = null;

  const closeList = (): void => {
    if (!listMode) {
      return;
    }
    html += `</${listMode}>`;
    listMode = null;
  };

  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed === "") {
      closeList();
      html += "<p class='blank'>&nbsp;</p>";
      continue;
    }

    const headingMatch = trimmed.match(/^(#{1,6})\s+(.+)$/);
    if (headingMatch) {
      closeList();
      const level = headingMatch[1].length;
      html += `<h${level}>${renderInline(headingMatch[2])}</h${level}>`;
      continue;
    }

    const ulMatch = trimmed.match(/^[-*]\s+(.+)$/);
    if (ulMatch) {
      if (listMode !== "ul") {
        closeList();
        html += "<ul>";
        listMode = "ul";
      }
      html += `<li>${renderInline(ulMatch[1])}</li>`;
      continue;
    }

    const olMatch = trimmed.match(/^\d+\.\s+(.+)$/);
    if (olMatch) {
      if (listMode !== "ol") {
        closeList();
        html += "<ol>";
        listMode = "ol";
      }
      html += `<li>${renderInline(olMatch[1])}</li>`;
      continue;
    }

    closeList();
    html += `<p>${renderInline(trimmed)}</p>`;
  }

  closeList();
  return html;
}

function renderInline(text: string): string {
  return text
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/\*([^*]+)\*/g, "<em>$1</em>")
    .replace(/\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>');
}

function escapeHtml(input: string): string {
  return input
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}
</script>
