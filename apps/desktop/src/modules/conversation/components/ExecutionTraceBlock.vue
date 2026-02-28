<template>
  <div class="message-row assistant trace-row">
    <div class="message-body trace-body">
      <div v-if="trace.isRunning" class="trace-running-list" role="status" aria-live="polite">
        <template v-if="runningActions.length > 0">
          <div
            v-for="action in runningActions"
            :key="action.actionId"
            class="trace-running-line"
          >
            <div class="trace-running-main">
              <span class="trace-running-label">{{ action.primary }}</span>
              <span class="trace-running-elapsed">{{ action.elapsedLabel }}</span>
            </div>
            <p class="trace-running-secondary">{{ action.secondary }}</p>
          </div>
        </template>
        <div v-else class="trace-running-line">
          <div class="trace-running-main">
            <span class="trace-running-label">{{ t("conversation.running.preparing") }}</span>
            <span class="trace-running-elapsed">0s</span>
          </div>
          <p class="trace-running-secondary">{{ t("conversation.running.secondary.pending") }}</p>
        </div>
      </div>

      <div class="trace-summary-head">
        <span class="trace-summary-text" :class="`trace-summary-tone-${trace.summaryTone}`">
          <span class="trace-summary-primary">{{ trace.summaryPrimary }}</span>
          <span v-if="trace.summarySecondary !== ''" class="trace-summary-secondary">{{ trace.summarySecondary }}</span>
        </span>
        <button
          class="trace-summary-toggle"
          type="button"
          :aria-expanded="trace.isExpanded ? 'true' : 'false'"
          :aria-controls="tracePanelId"
          :aria-label="trace.isExpanded ? t('conversation.trace.action.collapseAria') : t('conversation.trace.action.expandAria')"
          @click="toggleTrace(!trace.isExpanded)"
        >
          <span class="trace-summary-toggle-label">
            {{ trace.isExpanded ? t("conversation.trace.action.collapse") : t("conversation.trace.action.expand") }}
          </span>
          <AppIcon :name="trace.isExpanded ? 'chevron-up' : 'chevron-down'" :size="12" />
        </button>
      </div>

      <div v-if="trace.isExpanded" :id="tracePanelId" class="trace-disclosure-panel">
        <div class="trace-steps">
          <div v-for="step in trace.steps" :key="step.id" class="trace-step-line" :data-tone="step.statusTone">
            <div class="trace-step-main">
              <span class="trace-step-title">{{ step.title }}</span>
              <span class="trace-step-text">{{ step.summary }}</span>
              <span v-if="step.timestampLabel !== ''" class="trace-step-time">{{ step.timestampLabel }}</span>
            </div>
            <p v-if="step.detail !== ''" class="trace-step-detail">{{ step.detail }}</p>
            <details v-if="step.rawPayload !== ''" class="trace-step-raw">
              <summary class="trace-step-raw-summary">{{ t("conversation.trace.raw.expand") }}</summary>
              <pre class="trace-step-raw-content">{{ step.rawPayload }}</pre>
            </details>
          </div>
        </div>
        <div class="trace-panel-footer">
          <button
            class="trace-summary-toggle trace-summary-toggle-footer"
            type="button"
            :aria-label="t('conversation.trace.action.collapseAria')"
            @click="toggleTrace(false)"
          >
            <span class="trace-summary-toggle-label">{{ t("conversation.trace.action.collapse") }}</span>
            <AppIcon name="chevron-up" :size="12" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

import type { ExecutionTraceStep, ExecutionTraceViewModel } from "@/modules/conversation/views/processTrace";
import type { RunningActionViewModel } from "@/modules/conversation/views/runningActions";
import { useI18n } from "@/shared/i18n";
import AppIcon from "@/shared/ui/AppIcon.vue";

type ExecutionTrace = ExecutionTraceViewModel & { isExpanded: boolean; steps: ExecutionTraceStep[] };

const props = defineProps<{
  trace: ExecutionTrace;
  runningActions: RunningActionViewModel[];
}>();

const emit = defineEmits<{
  (event: "toggle-trace", executionId: string, expanded: boolean): void;
}>();

const { t } = useI18n();

const tracePanelId = computed(() => `trace-panel-${sanitizeExecutionId(props.trace.executionId)}`);

function toggleTrace(expanded: boolean): void {
  emit("toggle-trace", props.trace.executionId, expanded);
}

function sanitizeExecutionId(value: string): string {
  const normalized = value.trim().replace(/[^a-zA-Z0-9_-]/g, "-");
  return normalized !== "" ? normalized : "unknown";
}
</script>

<style scoped src="./ExecutionTraceBlock.css"></style>
