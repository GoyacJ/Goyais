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
            <span class="trace-running-label">{{ action.label }}</span>
            <span class="trace-running-elapsed">{{ action.elapsedLabel }}</span>
          </div>
        </template>
        <div v-else class="trace-running-line">
          <span class="trace-running-label">正在准备下一步</span>
          <span class="trace-running-elapsed">0s</span>
        </div>
      </div>

      <details class="trace-disclosure" :open="trace.isExpanded" @toggle="onTraceToggle">
        <summary class="trace-summary-inline">
          <span class="trace-caret">&gt;</span>
          <span class="trace-summary-text">执行过程：{{ trace.summary }}</span>
          <span class="trace-summary-action">{{ trace.isExpanded ? "收起详细过程" : "查看详细过程" }}</span>
        </summary>
        <div class="trace-steps">
          <div v-for="step in trace.steps" :key="step.id" class="trace-step-line">
            <div class="trace-step-main">
              <span class="trace-step-title">{{ step.title }}</span>
              <span class="trace-step-text">{{ step.summary }}</span>
            </div>
            <pre v-if="step.details !== ''" class="trace-step-details">{{ step.details }}</pre>
          </div>
        </div>
      </details>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ExecutionTraceStep, ExecutionTraceViewModel } from "@/modules/conversation/views/processTrace";
import type { RunningActionViewModel } from "@/modules/conversation/views/runningActions";

type ExecutionTrace = ExecutionTraceViewModel & { isExpanded: boolean; steps: ExecutionTraceStep[] };

const props = defineProps<{
  trace: ExecutionTrace;
  runningActions: RunningActionViewModel[];
}>();

const emit = defineEmits<{
  (event: "toggle-trace", executionId: string, expanded: boolean): void;
}>();

function onTraceToggle(event: Event): void {
  emit("toggle-trace", props.trace.executionId, (event.target as HTMLDetailsElement).open);
}
</script>

<style scoped src="./ExecutionTraceBlock.css"></style>
