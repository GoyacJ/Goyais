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

      <details class="trace-disclosure" :open="trace.isExpanded" @toggle="onTraceToggle">
        <summary class="trace-summary-inline">
          <span class="trace-caret">&gt;</span>
          <span class="trace-summary-text">
            <span class="trace-summary-primary">{{ trace.summaryPrimary }}</span>
            <span v-if="trace.summarySecondary !== ''" class="trace-summary-secondary">{{ trace.summarySecondary }}</span>
          </span>
          <span class="trace-summary-action">{{ trace.isExpanded ? t("conversation.trace.action.collapse") : t("conversation.trace.action.expand") }}</span>
        </summary>
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
      </details>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ExecutionTraceStep, ExecutionTraceViewModel } from "@/modules/conversation/views/processTrace";
import type { RunningActionViewModel } from "@/modules/conversation/views/runningActions";
import { useI18n } from "@/shared/i18n";

type ExecutionTrace = ExecutionTraceViewModel & { isExpanded: boolean; steps: ExecutionTraceStep[] };

const props = defineProps<{
  trace: ExecutionTrace;
  runningActions: RunningActionViewModel[];
}>();

const emit = defineEmits<{
  (event: "toggle-trace", executionId: string, expanded: boolean): void;
}>();

const { t } = useI18n();

function onTraceToggle(event: Event): void {
  emit("toggle-trace", props.trace.executionId, (event.target as HTMLDetailsElement).open);
}
</script>

<style scoped src="./ExecutionTraceBlock.css"></style>
