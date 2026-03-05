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
            <div v-if="message.role === 'user'" class="message-actions">
              <button
                v-if="message.can_rollback"
                class="rollback"
                type="button"
                @click="$emit('rollback', message.id)"
              >
                {{ t("session.message.rollback") }}
              </button>
              <button
                class="rollback copy"
                type="button"
                @click="copyUserMessage(message.content)"
              >
                {{ t("session.message.copy") }}
              </button>
            </div>
          </div>
        </div>

        <div
          v-for="trace in getMessageTraces(message)"
          :key="trace.executionId"
          class="trace-host"
        >
          <RunTraceBlock
            :trace="trace"
            :running-actions="getRunningActions(trace.executionId)"
            @select-trace="onTraceSelect"
          />
        </div>

        <div
          v-for="question in getMessagePendingQuestions(message)"
          :key="`${question.executionId}:${question.questionId}`"
          class="question-card"
        >
          <p class="question-title">{{ question.question }}</p>
          <div v-if="question.options.length > 0" class="question-options" role="radiogroup">
            <label
              v-for="option in question.options"
              :key="option.id"
              class="question-option"
            >
              <input
                type="radio"
                :name="`question-${question.executionId}-${question.questionId}`"
                :checked="getQuestionDraft(question).selectedOptionId === option.id"
                @change="onQuestionOptionChange(question, option.id)"
              />
              <span class="question-option-main">
                <span class="question-option-label">
                  {{ option.label }}
                  <span
                    v-if="question.recommendedOptionId !== '' && question.recommendedOptionId === option.id"
                    class="question-option-recommended"
                  >
                    Recommended
                  </span>
                </span>
                <span v-if="option.description !== ''" class="question-option-description">{{ option.description }}</span>
              </span>
            </label>
          </div>
          <textarea
            v-if="question.allowText"
            class="question-text"
            :value="getQuestionDraft(question).text"
            placeholder="补充说明（可选）"
            @input="onQuestionTextInput(question, $event)"
          ></textarea>
          <div class="question-actions">
            <button
              class="question-submit"
              type="button"
              :disabled="!canSubmitQuestion(question)"
              @click="submitQuestion(question)"
            >
              提交回答
            </button>
          </div>
        </div>
      </template>
    </div>

    <div class="composer" :class="`composer-mode-${mode}`">
      <div
        v-if="queuedMessages.length > 0"
        class="queued-list"
        role="list"
        :aria-label="t('session.queue.listAria')"
      >
        <div v-for="queuedMessage in queuedMessages" :key="queuedMessage.executionId" class="queued-item" role="listitem">
          <div class="queued-main">
            <AppIcon name="corner-down-right" :size="12" />
            <p class="queued-text" :title="queuedMessage.content.trim()">
              {{ queuedMessage.preview !== "" ? queuedMessage.preview : t("session.queue.itemFallback") }}
            </p>
          </div>
          <button
            class="queued-remove"
            type="button"
            :aria-label="t('session.queue.removeAria')"
            @click="onRemoveQueued(queuedMessage.executionId)"
          >
            <AppIcon name="trash-2" :size="12" />
          </button>
        </div>
      </div>

      <div class="draft-wrap">
        <textarea
          ref="draftRef"
          class="draft"
          :value="draft"
          :placeholder="placeholder"
          @input="onDraftInput"
          @keydown="onDraftKeydown"
          @keyup="onDraftKeyup"
          @click="onDraftClick"
        ></textarea>
        <div ref="suggestionListRef" v-if="showSuggestionPanel" class="suggestions" role="listbox" aria-label="输入候选">
          <button
            v-for="(suggestion, index) in composerSuggestions"
            :key="`${suggestion.kind}-${suggestion.insert_text}-${index}`"
            class="suggestion-item"
            :class="{ active: index === activeSuggestionIndex }"
            type="button"
            role="option"
            :aria-selected="index === activeSuggestionIndex"
            :ref="(el) => setSuggestionItemRef(index, el as HTMLButtonElement | null)"
            @mousedown.prevent="applySuggestionFromIndex(index)"
            @mouseenter="updateActiveSuggestionIndex(index)"
          >
            <span class="suggestion-content">
              <span class="suggestion-label">{{ suggestion.label }}</span>
              <span v-if="hasSuggestionDetail(suggestion)" class="suggestion-meta">{{ suggestion.detail }}</span>
            </span>
          </button>
          <div v-if="composerSuggesting && composerSuggestions.length === 0" class="suggestion-empty">
            {{ t("session.composer.suggestion.loading") }}
          </div>
        </div>
      </div>

      <div class="composer-actions">
        <div class="left">
          <div ref="plusWrapRef" class="plus-wrap">
            <button class="action-btn" type="button" aria-label="更多操作" @click="plusOpen = !plusOpen">
              <AppIcon name="plus" :size="14" />
            </button>
            <div v-if="plusOpen" ref="plusMenuRef" class="plus-menu">
              <button
                v-for="(item, index) in plusMenuItems"
                :key="`plus-item-${item.id}`"
                type="button"
                class="plus-menu-item"
                :data-menu-id="item.id"
                :class="{ active: plusMenuActiveIndex === index, expanded: item.id === 'advancedMode' && plusAdvancedModeOpen }"
                @mouseenter="onPlusMenuItemHover(index)"
                @click="onPlusMenuItemClick(item.id)"
              >
                <span>{{ item.label }}</span>
                <span v-if="item.id === 'advancedMode'" class="plus-submenu-caret">›</span>
              </button>
              <div
                v-if="plusAdvancedModeOpen"
                ref="plusSubmenuRef"
                class="plus-submenu"
                :class="plusSubmenuPlacement === 'left' ? 'plus-submenu--left' : 'plus-submenu--right'"
              >
                <button
                  v-for="(item, index) in permissionModeOptions"
                  :key="`plus-mode-${item.id}`"
                  type="button"
                  class="plus-mode-item"
                  :data-mode-id="item.id"
                  :class="{ active: plusModeActiveIndex === index, selected: mode === item.id, danger: item.dangerous }"
                  @mouseenter="onPlusModeItemHover(index)"
                  @click="onPlusModeItemClick(index)"
                >
                  <span>{{ item.label }}</span>
                  <span v-if="mode === item.id" class="plus-mode-current">{{ t("session.mode.current") }}</span>
                </button>
              </div>
            </div>
          </div>

          <select class="select" :value="quickModeValue" @change="onQuickModeChange">
            <option v-if="isAdvancedModeActive" :value="quickModeAdvancedValue" disabled>
              {{ quickModeAdvancedLabel }}
            </option>
            <option value="default">{{ getPermissionModeLabel("default") }}</option>
            <option value="plan">{{ getPermissionModeLabel("plan") }}</option>
          </select>
          <select class="select" :value="modelId" :disabled="!hasModelOptions" @change="onModelChange">
            <option v-if="!hasModelOptions" value="">无可用模型</option>
            <option v-for="option in modelSelectOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </div>

        <div class="right">
          <button
            v-if="hasConfirmingExecution"
            class="action-btn approval approve"
            type="button"
            aria-label="批准工具调用"
            @click="$emit('approve')"
          >
            批准
          </button>
          <button
            v-if="hasConfirmingExecution"
            class="action-btn approval deny"
            type="button"
            aria-label="拒绝工具调用"
            @click="$emit('deny')"
          >
            拒绝
          </button>
          <button
            class="action-btn primary-action"
            :class="{ send: !shouldShowStopAction }"
            type="button"
            :aria-label="primaryActionAriaLabel"
            :disabled="isPrimaryActionDisabled"
            @click="onPrimaryAction"
          >
            <AppIcon :name="shouldShowStopAction ? 'square' : 'arrow-up'" :size="12" />
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, onUpdated, ref, watch } from "vue";
import {
  type ComposerSuggestion,
  type SessionMessage,
  type ConversationMode
} from "@/shared/types/api";
import type { RunTraceViewModel } from "@/modules/session/views/processTrace";
import type { RunningActionViewModel } from "@/modules/session/views/runningActions";
import RunTraceBlock from "@/modules/session/components/RunTraceBlock.vue";
import AppIcon from "@/shared/ui/AppIcon.vue";
import { useI18n } from "@/shared/i18n";
import { showToast } from "@/shared/stores/toastStore";

type TraceViewModel = RunTraceViewModel;
type QueuedMessageViewModel = {
  executionId: string;
  messageId: string;
  queueIndex: number;
  content: string;
  preview: string;
};
type PendingQuestionOption = {
  id: string;
  label: string;
  description: string;
};
type PendingQuestionViewModel = {
  executionId: string;
  messageId: string;
  queueIndex: number;
  questionId: string;
  question: string;
  options: PendingQuestionOption[];
  recommendedOptionId: string;
  allowText: boolean;
  required: boolean;
};
type PendingQuestionDraft = {
  selectedOptionId: string;
  text: string;
};
type PlusMenuItemID = "addResource" | "insertCommand" | "advancedMode";
type PlusMenuItem = {
  id: PlusMenuItemID;
  label: string;
};
type PlusSubmenuPlacement = "right" | "left";

const props = withDefaults(
  defineProps<{
    messages: SessionMessage[];
    queuedMessages?: QueuedMessageViewModel[];
    queuedCount: number;
    pendingCount: number;
    executingCount: number;
    hasActiveExecution: boolean;
    hasConfirmingExecution?: boolean;
    pendingQuestions?: PendingQuestionViewModel[];
    activeTraceCount: number;
    executionTraces: TraceViewModel[];
    runningActions: RunningActionViewModel[];
    draft: string;
    mode: ConversationMode;
    modelId: string;
    placeholder: string;
    composerSuggestions?: ComposerSuggestion[];
    composerSuggesting?: boolean;
    modelOptions?: Array<{ value: string; label: string }>;
  }>(),
  {
    modelOptions: () => [],
    queuedMessages: () => [],
    composerSuggestions: () => [],
    composerSuggesting: false,
    hasConfirmingExecution: false,
    pendingQuestions: () => []
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
const executionTracesByMessageId = computed(() => {
  const mapped = new Map<string, TraceViewModel[]>();
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
  const mapped = new Map<number, TraceViewModel[]>();
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
  (event: "remove-queued", executionID: string): void;
  (event: "approve"): void;
  (event: "deny"): void;
  (event: "rollback", messageId: string): void;
  (event: "select-trace", executionId: string): void;
  (event: "answer-question", payload: { executionId: string; questionId: string; selectedOptionId?: string; text?: string }): void;
  (event: "update:draft", value: string): void;
  (event: "update:mode", value: ConversationMode): void;
  (event: "update:model", value: string): void;
  (event: "request-suggestions", input: { draft: string; cursor: number }): void;
  (event: "clear-suggestions"): void;
}>();

const { t } = useI18n();
const plusOpen = ref(false);
const plusWrapRef = ref<HTMLElement | null>(null);
const plusMenuRef = ref<HTMLElement | null>(null);
const plusSubmenuRef = ref<HTMLElement | null>(null);
const plusSubmenuPlacement = ref<PlusSubmenuPlacement>("right");
const draftRef = ref<HTMLTextAreaElement | null>(null);
const conversationAreaRef = ref<HTMLElement | null>(null);
const suggestionListRef = ref<HTMLElement | null>(null);
const shouldStickToBottom = ref(true);
const activeSuggestionIndex = ref(0);
const suggestionItemRefs = ref<Array<HTMLButtonElement | null>>([]);
const showSuggestionPanel = computed(() => props.composerSuggestions.length > 0 || props.composerSuggesting);
const shouldShowStopAction = computed(() => props.hasActiveExecution && props.draft.trim() === "");
const isPrimaryActionDisabled = computed(() => (shouldShowStopAction.value ? !props.hasActiveExecution : !hasModelOptions.value));
const primaryActionAriaLabel = computed(() => (shouldShowStopAction.value ? t("session.stop") : t("session.send")));
const permissionModeOptions = computed<Array<{ id: ConversationMode; label: string; dangerous: boolean }>>(() => [
  { id: "default", label: t("session.mode.option.default"), dangerous: false },
  { id: "plan", label: t("session.mode.option.plan"), dangerous: false },
  { id: "acceptEdits", label: t("session.mode.option.acceptEdits"), dangerous: false },
  { id: "dontAsk", label: t("session.mode.option.dontAsk"), dangerous: true },
  { id: "bypassPermissions", label: t("session.mode.option.bypassPermissions"), dangerous: true }
]);
const plusMenuActiveIndex = ref(0);
const plusModeActiveIndex = ref(0);
const plusAdvancedModeOpen = ref(false);
const plusMenuItems = computed<PlusMenuItem[]>(() => [
  { id: "addResource", label: t("session.composer.menu.addResource") },
  { id: "insertCommand", label: t("session.composer.menu.insertCommand") },
  { id: "advancedMode", label: t("session.composer.menu.advancedMode") }
]);
const isAdvancedModeActive = computed(() => props.mode !== "default" && props.mode !== "plan");
const quickModeAdvancedValue = computed(() => `advanced:${props.mode}`);
const quickModeAdvancedLabel = computed(() => {
  return `${t("session.mode.quick.advancedPrefix")} · ${getPermissionModeLabel(props.mode)}`;
});
const quickModeValue = computed<string>(() => {
  if (isAdvancedModeActive.value) {
    return quickModeAdvancedValue.value;
  }
  return props.mode === "plan" ? "plan" : "default";
});
const pendingQuestionDrafts = ref<Record<string, PendingQuestionDraft>>({});

watch(
  () => props.composerSuggestions.length,
  () => {
    suggestionItemRefs.value = [];
    if (props.composerSuggestions.length <= 0) {
      activeSuggestionIndex.value = 0;
    } else if (activeSuggestionIndex.value >= props.composerSuggestions.length) {
      activeSuggestionIndex.value = 0;
    }
  }
);

watch(
  plusOpen,
  (opened) => {
    if (!opened) {
      plusAdvancedModeOpen.value = false;
      plusSubmenuPlacement.value = "right";
      return;
    }
    plusMenuActiveIndex.value = 0;
    plusAdvancedModeOpen.value = false;
    plusModeActiveIndex.value = getCurrentModeIndex();
  }
);
watch(plusAdvancedModeOpen, (opened) => {
  if (!opened) {
    plusSubmenuPlacement.value = "right";
    return;
  }
  void nextTick(() => {
    updatePlusSubmenuPlacement();
  });
});

watch(
  () => props.pendingQuestions,
  (questions) => {
    const nextDrafts: Record<string, PendingQuestionDraft> = {};
    for (const question of questions) {
      const key = pendingQuestionKey(question);
      const previous = pendingQuestionDrafts.value[key];
      const fallbackSelected = question.options.some((item) => item.id === question.recommendedOptionId)
        ? question.recommendedOptionId
        : "";
      nextDrafts[key] = {
        selectedOptionId: previous?.selectedOptionId ?? fallbackSelected,
        text: previous?.text ?? ""
      };
    }
    pendingQuestionDrafts.value = nextDrafts;
  },
  { immediate: true, deep: true }
);

function updateActiveSuggestionIndex(next: number): void {
  const total = props.composerSuggestions.length;
  if (total <= 0) {
    activeSuggestionIndex.value = 0;
    return;
  }
  const normalized = ((next % total) + total) % total;
  activeSuggestionIndex.value = normalized;
  void nextTick(() => {
    const element = suggestionItemRefs.value[normalized];
    if (!element || !suggestionListRef.value) {
      return;
    }
    element.scrollIntoView({ block: "nearest" });
  });
}

function setSuggestionItemRef(index: number, element: HTMLButtonElement | null): void {
  suggestionItemRefs.value[index] = element;
}

function onDraftInput(event: Event): void {
  const element = event.target as HTMLTextAreaElement;
  emit("update:draft", element.value);
  requestComposerSuggestions(element);
}

function onDraftClick(event: Event): void {
  requestComposerSuggestions(event.target as HTMLTextAreaElement);
}

function onDraftKeyup(event: KeyboardEvent): void {
  const ignoredKeys = new Set(["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Enter", "Escape", "Tab"]);
  if (ignoredKeys.has(event.key)) {
    return;
  }
  requestComposerSuggestions(event.target as HTMLTextAreaElement);
}

function onQuickModeChange(event: Event): void {
  const nextValue = (event.target as HTMLSelectElement).value;
  if (!isQuickModeValue(nextValue)) {
    return;
  }
  requestModeChange(nextValue);
}

function onAdvancedModeSelect(mode: ConversationMode): void {
  requestModeChange(mode);
}

function requestModeChange(nextMode: ConversationMode): void {
  if (nextMode === props.mode) {
    plusOpen.value = false;
    return;
  }
  if (isDangerousMode(nextMode)) {
    const modeLabel = getPermissionModeLabel(nextMode);
    const confirmMessage = `${t("session.mode.confirmDangerous")} (${modeLabel})`;
    const confirmed = typeof window === "undefined"
      ? true
      : window.confirm(confirmMessage);
    if (!confirmed) {
      return;
    }
  }
  emit("update:mode", nextMode);
  plusOpen.value = false;
}

function onPlusMenuItemHover(index: number): void {
  if (index < 0 || index >= plusMenuItems.value.length) {
    return;
  }
  plusMenuActiveIndex.value = index;
  if (plusMenuItems.value[index]?.id === "advancedMode") {
    if (!plusAdvancedModeOpen.value) {
      openPlusAdvancedMode();
    }
    return;
  }
  if (plusMenuItems.value[index]?.id !== "advancedMode") {
    plusAdvancedModeOpen.value = false;
  }
}

function onPlusMenuItemClick(itemID: PlusMenuItemID): void {
  switch (itemID) {
    case "addResource":
      insertResourceToken();
      return;
    case "insertCommand":
      insertCommandToken();
      return;
    case "advancedMode":
      openPlusAdvancedMode();
      return;
    default:
      return;
  }
}

function onPlusModeItemHover(index: number): void {
  if (index < 0 || index >= permissionModeOptions.value.length) {
    return;
  }
  plusModeActiveIndex.value = index;
}

function onPlusModeItemClick(index: number): void {
  const option = permissionModeOptions.value[index];
  if (!option) {
    return;
  }
  requestModeChange(option.id);
}

function openPlusAdvancedMode(): void {
  plusAdvancedModeOpen.value = true;
  plusSubmenuPlacement.value = "right";
  plusModeActiveIndex.value = getCurrentModeIndex();
  void nextTick(() => {
    updatePlusSubmenuPlacement();
  });
}

function getCurrentModeIndex(): number {
  const index = permissionModeOptions.value.findIndex((item) => item.id === props.mode);
  return index >= 0 ? index : 0;
}

function getPermissionModeLabel(mode: ConversationMode): string {
  const option = permissionModeOptions.value.find((item) => item.id === mode);
  return option?.label ?? mode;
}

function onModelChange(event: Event): void {
  emit("update:model", (event.target as HTMLSelectElement).value);
}

function getRunningActions(executionId: string): RunningActionViewModel[] {
  return runningActionsByExecutionId.value.get(executionId) ?? [];
}

function getMessageTraces(message: SessionMessage): TraceViewModel[] {
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
  const merged = new Map<string, TraceViewModel>();
  for (const item of directMatches) {
    merged.set(item.executionId, item);
  }
  for (const item of queueMatches) {
    merged.set(item.executionId, item);
  }
  return [...merged.values()];
}

function getMessagePendingQuestions(message: SessionMessage): PendingQuestionViewModel[] {
  const directMatches = props.pendingQuestions.filter((item) => item.messageId === message.id);
  if (directMatches.length > 0) {
    return directMatches;
  }
  if (typeof message.queue_index !== "number") {
    return [];
  }
  return props.pendingQuestions.filter((item) => item.queueIndex === message.queue_index);
}

function onTraceSelect(executionId: string): void {
  emit("select-trace", executionId);
}

async function copyUserMessage(content: string): Promise<void> {
  const value = content.trim();
  if (value === "") {
    return;
  }
  try {
    await copyTextToClipboard(value);
    showToast({
      tone: "success",
      message: t("session.message.copySuccess")
    });
  } catch {
    showToast({
      tone: "error",
      message: t("session.message.copyFailed")
    });
  }
}

function onPrimaryAction(): void {
  if (shouldShowStopAction.value) {
    emit("stop");
    return;
  }
  emit("send");
}

function onRemoveQueued(executionID: string): void {
  emit("remove-queued", executionID);
}

function onDraftKeydown(event: KeyboardEvent): void {
  if (
    event.key === "Tab" &&
    event.shiftKey &&
    !event.ctrlKey &&
    !event.altKey &&
    !event.metaKey
  ) {
    event.preventDefault();
    requestModeChange(quickModeValue.value === "plan" ? "default" : "plan");
    return;
  }

  if (showSuggestionPanel.value && props.composerSuggestions.length > 0) {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      updateActiveSuggestionIndex(activeSuggestionIndex.value + 1);
      return;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      updateActiveSuggestionIndex(activeSuggestionIndex.value - 1);
      return;
    }
    if (event.key === "Tab") {
      event.preventDefault();
      applySuggestionFromIndex(activeSuggestionIndex.value);
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      emit("clear-suggestions");
      return;
    }
  }

  if (event.key !== "Enter") {
    return;
  }
  if (event.shiftKey || event.ctrlKey || event.altKey || event.metaKey || event.isComposing) {
    return;
  }
  if (showSuggestionPanel.value && props.composerSuggestions.length > 0) {
    event.preventDefault();
    applySuggestionFromIndex(activeSuggestionIndex.value);
    return;
  }
  event.preventDefault();
  emit("send");
}

function getQuestionDraft(question: PendingQuestionViewModel): PendingQuestionDraft {
  return pendingQuestionDrafts.value[pendingQuestionKey(question)] ?? {
    selectedOptionId: "",
    text: ""
  };
}

function onQuestionOptionChange(question: PendingQuestionViewModel, optionID: string): void {
  const key = pendingQuestionKey(question);
  const current = getQuestionDraft(question);
  pendingQuestionDrafts.value = {
    ...pendingQuestionDrafts.value,
    [key]: {
      ...current,
      selectedOptionId: optionID
    }
  };
}

function onQuestionTextInput(question: PendingQuestionViewModel, event: Event): void {
  const key = pendingQuestionKey(question);
  const current = getQuestionDraft(question);
  pendingQuestionDrafts.value = {
    ...pendingQuestionDrafts.value,
    [key]: {
      ...current,
      text: (event.target as HTMLTextAreaElement).value
    }
  };
}

function canSubmitQuestion(question: PendingQuestionViewModel): boolean {
  const draft = getQuestionDraft(question);
  const selectedOptionID = draft.selectedOptionId.trim();
  const text = draft.text.trim();
  if (text !== "" && !question.allowText) {
    return false;
  }
  if (selectedOptionID !== "") {
    return question.options.length === 0 || question.options.some((item) => item.id === selectedOptionID);
  }
  if (text !== "") {
    return true;
  }
  return !question.required;
}

function submitQuestion(question: PendingQuestionViewModel): void {
  if (!canSubmitQuestion(question)) {
    return;
  }
  const draft = getQuestionDraft(question);
  const selectedOptionID = draft.selectedOptionId.trim();
  const text = draft.text.trim();
  emit("answer-question", {
    executionId: question.executionId,
    questionId: question.questionId,
    selectedOptionId: selectedOptionID !== "" ? selectedOptionID : undefined,
    text: text !== "" ? text : undefined
  });
}

function requestComposerSuggestions(element: HTMLTextAreaElement | null): void {
  if (!element) {
    emit("clear-suggestions");
    return;
  }
  const draft = element.value;
  const cursor = element.selectionStart ?? draft.length;
  emit("request-suggestions", { draft, cursor });
  activeSuggestionIndex.value = 0;
}

function applySuggestionFromIndex(index: number): void {
  const suggestion = props.composerSuggestions[index];
  if (!suggestion) {
    return;
  }
  const isResourceTypeSuggestion = suggestion.kind === "resource_type";
  const currentDraft = props.draft;
  const replaceStart = Math.max(0, Math.min(suggestion.replace_start, currentDraft.length));
  const replaceEnd = Math.max(replaceStart, Math.min(suggestion.replace_end, currentDraft.length));
  const replaced = `${currentDraft.slice(0, replaceStart)}${suggestion.insert_text}${currentDraft.slice(replaceEnd)}`;
  const nextDraft = isResourceTypeSuggestion
    ? replaced
    : /[\s\n]$/.test(replaced)
      ? replaced
      : `${replaced} `;
  emit("update:draft", nextDraft);
  activeSuggestionIndex.value = 0;
  if (!isResourceTypeSuggestion) {
    emit("clear-suggestions");
  }
  void nextTick(() => {
    const element = draftRef.value;
    if (!element) {
      return;
    }
    const caret = Math.min(
      nextDraft.length,
      replaceStart + suggestion.insert_text.length + (isResourceTypeSuggestion ? 0 : 1)
    );
    element.focus();
    element.setSelectionRange(caret, caret);
    if (isResourceTypeSuggestion) {
      emit("request-suggestions", { draft: nextDraft, cursor: caret });
    }
  });
}

function insertResourceToken(): void {
  plusOpen.value = false;
  insertTokenAtCaret("@");
}

function insertCommandToken(): void {
  plusOpen.value = false;
  insertTokenAtCaret("/");
}

function insertTokenAtCaret(token: string): void {
  const element = draftRef.value;
  if (!element) {
    return;
  }
  const start = element.selectionStart ?? props.draft.length;
  const end = element.selectionEnd ?? start;
  const nextDraft = `${props.draft.slice(0, start)}${token}${props.draft.slice(end)}`;
  emit("update:draft", nextDraft);
  void nextTick(() => {
    const nextCaret = start + token.length;
    element.focus();
    element.setSelectionRange(nextCaret, nextCaret);
    emit("request-suggestions", { draft: nextDraft, cursor: nextCaret });
  });
}

function hasSuggestionDetail(suggestion: ComposerSuggestion): boolean {
  return (suggestion.detail ?? "").trim() !== "";
}

function pendingQuestionKey(question: PendingQuestionViewModel): string {
  return `${question.executionId}:${question.questionId}`;
}

function isDangerousMode(mode: ConversationMode): boolean {
  return mode === "dontAsk" || mode === "bypassPermissions";
}

function isQuickModeValue(value: string): value is "default" | "plan" {
  return value === "default" || value === "plan";
}

function updatePlusSubmenuPlacement(): void {
  if (!plusOpen.value || !plusAdvancedModeOpen.value) {
    return;
  }
  if (typeof window === "undefined") {
    plusSubmenuPlacement.value = "right";
    return;
  }
  const menuElement = plusMenuRef.value;
  const submenuElement = plusSubmenuRef.value;
  if (!menuElement || !submenuElement) {
    plusSubmenuPlacement.value = "right";
    return;
  }

  const gap = 8;
  const menuRect = menuElement.getBoundingClientRect();
  const submenuWidth = submenuElement.getBoundingClientRect().width;
  const roomRight = window.innerWidth - menuRect.right - gap;
  const roomLeft = menuRect.left - gap;

  if (roomRight >= submenuWidth) {
    plusSubmenuPlacement.value = "right";
    return;
  }
  if (roomLeft >= submenuWidth) {
    plusSubmenuPlacement.value = "left";
    return;
  }
  plusSubmenuPlacement.value = roomLeft > roomRight ? "left" : "right";
}

function onGlobalResize(): void {
  if (!plusOpen.value || !plusAdvancedModeOpen.value) {
    return;
  }
  updatePlusSubmenuPlacement();
}

async function copyTextToClipboard(text: string): Promise<void> {
  if (typeof navigator !== "undefined" && navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
    await navigator.clipboard.writeText(text);
    return;
  }
  if (typeof document === "undefined") {
    throw new Error("Clipboard API unavailable");
  }
  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.setAttribute("readonly", "true");
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  const copied = document.execCommand("copy");
  document.body.removeChild(textarea);
  if (!copied) {
    throw new Error("Clipboard copy failed");
  }
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

function onGlobalPointerDown(event: PointerEvent): void {
  const target = event.target;
  if (!(target instanceof Node)) {
    return;
  }
  if (plusOpen.value && plusWrapRef.value && !plusWrapRef.value.contains(target)) {
    plusOpen.value = false;
  }
}

function onGlobalKeyDown(event: KeyboardEvent): void {
  if (!plusOpen.value) {
    return;
  }

  const { key } = event;
  if (
    key !== "ArrowUp" &&
    key !== "ArrowDown" &&
    key !== "ArrowRight" &&
    key !== "ArrowLeft" &&
    key !== "Enter" &&
    key !== "Escape"
  ) {
    return;
  }

  event.preventDefault();

  if (key === "Escape") {
    plusAdvancedModeOpen.value = false;
    plusOpen.value = false;
    return;
  }

  if (key === "ArrowLeft") {
    if (plusAdvancedModeOpen.value) {
      plusAdvancedModeOpen.value = false;
    }
    return;
  }

  if (key === "ArrowDown") {
    if (plusAdvancedModeOpen.value) {
      plusModeActiveIndex.value = (plusModeActiveIndex.value + 1) % permissionModeOptions.value.length;
    } else {
      plusMenuActiveIndex.value = (plusMenuActiveIndex.value + 1) % plusMenuItems.value.length;
      if (plusMenuItems.value[plusMenuActiveIndex.value]?.id !== "advancedMode") {
        plusAdvancedModeOpen.value = false;
      }
    }
    return;
  }

  if (key === "ArrowUp") {
    if (plusAdvancedModeOpen.value) {
      plusModeActiveIndex.value = (
        plusModeActiveIndex.value - 1 + permissionModeOptions.value.length
      ) % permissionModeOptions.value.length;
    } else {
      plusMenuActiveIndex.value = (plusMenuActiveIndex.value - 1 + plusMenuItems.value.length) % plusMenuItems.value.length;
      if (plusMenuItems.value[plusMenuActiveIndex.value]?.id !== "advancedMode") {
        plusAdvancedModeOpen.value = false;
      }
    }
    return;
  }

  if (key === "ArrowRight") {
    if (plusAdvancedModeOpen.value) {
      onPlusModeItemClick(plusModeActiveIndex.value);
      return;
    }
    const activeItem = plusMenuItems.value[plusMenuActiveIndex.value];
    if (activeItem?.id === "advancedMode") {
      openPlusAdvancedMode();
    }
    return;
  }

  if (key === "Enter") {
    if (plusAdvancedModeOpen.value) {
      onPlusModeItemClick(plusModeActiveIndex.value);
      return;
    }
    const activeItem = plusMenuItems.value[plusMenuActiveIndex.value];
    if (activeItem) {
      onPlusMenuItemClick(activeItem.id);
    }
  }
}

onMounted(() => {
  if (typeof window !== "undefined") {
    window.addEventListener("pointerdown", onGlobalPointerDown);
    window.addEventListener("keydown", onGlobalKeyDown);
    window.addEventListener("resize", onGlobalResize);
  }
  void nextTick(scrollConversationToBottom);
});

onBeforeUnmount(() => {
  if (typeof window !== "undefined") {
    window.removeEventListener("pointerdown", onGlobalPointerDown);
    window.removeEventListener("keydown", onGlobalKeyDown);
    window.removeEventListener("resize", onGlobalResize);
  }
});

onUpdated(() => {
  if (!shouldStickToBottom.value) {
    return;
  }
  void nextTick(scrollConversationToBottom);
});
</script>

<style scoped src="./MainConversationPanel.css"></style>
