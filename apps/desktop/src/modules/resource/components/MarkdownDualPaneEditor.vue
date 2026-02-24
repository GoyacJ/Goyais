<template>
  <div class="editor-wrap">
    <div class="pane">
      <p v-if="label !== ''" class="pane-title">{{ label }}</p>
      <textarea
        class="editor"
        :placeholder="placeholder"
        :value="modelValue"
        :disabled="disabled"
        @input="onInput"
      />
    </div>

    <div class="pane">
      <p class="pane-title">实时预览</p>
      <article class="preview" v-html="renderedHtml" />
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

<style scoped>
.editor-wrap {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--global-space-10);
}

.pane {
  display: grid;
  gap: var(--global-space-6);
}

.pane-title {
  margin: 0;
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
}

.editor {
  min-height: 220px;
  resize: vertical;
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  color: var(--semantic-text);
  padding: var(--global-space-10);
  font-size: var(--global-font-size-13);
  font-family: var(--global-font-family-code);
}

.preview {
  min-height: 220px;
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  color: var(--semantic-text);
  padding: var(--global-space-10);
  overflow: auto;
}

.preview :deep(h1),
.preview :deep(h2),
.preview :deep(h3),
.preview :deep(h4),
.preview :deep(h5),
.preview :deep(h6),
.preview :deep(p) {
  margin: 0 0 var(--global-space-8) 0;
}

.preview :deep(ul),
.preview :deep(ol) {
  margin: 0 0 var(--global-space-8) 0;
  padding-left: var(--global-space-20);
}

.preview :deep(code) {
  background: var(--semantic-surface-2);
  border-radius: var(--global-radius-4);
  padding: 0 var(--global-space-4);
  font-family: var(--global-font-family-code);
}

.preview :deep(a) {
  color: var(--semantic-link);
}

@media (max-width: 960px) {
  .editor-wrap {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
