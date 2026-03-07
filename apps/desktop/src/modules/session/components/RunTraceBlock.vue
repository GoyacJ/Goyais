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
            <span class="trace-running-label">{{ t("session.running.preparing") }}</span>
            <span class="trace-running-elapsed">0s</span>
          </div>
          <p class="trace-running-secondary">{{ t("session.running.secondary.pending") }}</p>
        </div>
      </div>

      <button
        class="trace-summary-brief"
        type="button"
        :aria-label="t('session.trace.action.expandAria')"
        @click="selectTrace"
      >
        <span class="trace-summary-text" :class="`trace-summary-tone-${trace.summaryTone}`">
          <span class="trace-summary-primary">{{ trace.summaryPrimary }}</span>
          <span v-if="trace.summarySecondary !== ''" class="trace-summary-secondary">{{ trace.summarySecondary }}</span>
        </span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { RunTraceViewModel } from "@/modules/session/views/processTrace";
import type { RunningActionViewModel } from "@/modules/session/views/runningActions";
import { useI18n } from "@/shared/i18n";

const props = defineProps<{
  trace: RunTraceViewModel;
  runningActions: RunningActionViewModel[];
}>();

const emit = defineEmits<{
  (event: "select-trace", executionId: string): void;
}>();

const { t } = useI18n();

function selectTrace(): void {
  emit("select-trace", props.trace.executionId);
}
</script>

<style scoped src="./RunTraceBlock.css"></style>
