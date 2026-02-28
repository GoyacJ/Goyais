<template>
  <aside class="inspector">
    <div class="head">
      <p class="title">{{ t("conversation.inspector.title") }}</p>
      <button class="collapse-btn" type="button" :title="t('conversation.inspector.action.collapse')" @click="$emit('toggleCollapse')">
        <AppIcon name="panel-right-close" :size="12" />
      </button>
    </div>

    <div class="tabs">
      <button
        v-for="item in tabs"
        :key="item.key"
        class="tab"
        :class="{ active: item.key === activeTab }"
        type="button"
        @click="$emit('changeTab', item.key)"
      >
        {{ item.label }}
      </button>
    </div>

    <section v-if="activeTab === 'diff'" class="card">
      <div class="card-head">
        <strong>{{ t("conversation.inspector.tab.diff") }}</strong>
        <span>{{ tf("conversation.inspector.diff.filesCount", { count: diff.length }) }}</span>
      </div>

      <div class="diff-list">
        <div v-for="item in diff" :key="item.id" class="diff-row">
          <span class="path">{{ item.path }}</span>
          <span class="stat" :class="item.change_type">{{ mapChange(item.change_type) }}</span>
        </div>
      </div>

      <div class="actions">
        <button class="action" type="button" :disabled="!capability.can_commit" @click="$emit('commit')">
          {{ t("conversation.inspector.action.commit") }}
        </button>
        <button class="action" type="button" :disabled="!capability.can_discard" @click="$emit('discard')">
          {{ t("conversation.inspector.action.discard") }}
        </button>
        <button class="action" type="button" :disabled="!capability.can_export_patch" @click="$emit('exportPatch')">
          {{ t("conversation.inspector.action.exportPatch") }}
        </button>
      </div>
      <p v-if="capability.reason" class="reason">{{ capability.reason }}</p>
    </section>

    <section v-else-if="activeTab === 'run'" class="card">
      <strong>{{ t("conversation.inspector.tab.run") }}</strong>
      <p>
        {{
          tf("conversation.inspector.run.counts", {
            pending: pendingCount,
            executing: executingCount,
            queued: queuedCount
          })
        }}
      </p>
      <p v-if="latestExecutionLabel" class="normal">{{ latestExecutionLabel }}</p>
      <p v-if="latestExecutionMetrics" class="normal">{{ latestExecutionMetrics }}</p>
      <p :class="runHintTone">{{ runHint }}</p>
    </section>

    <section v-else-if="activeTab === 'trace'" class="card trace-card">
      <strong>{{ t("conversation.inspector.tab.trace") }}</strong>
      <p v-if="!selectedTrace" class="normal">{{ t("conversation.inspector.trace.empty") }}</p>
      <template v-else>
        <p class="normal">{{ tf("conversation.inspector.trace.execution", { id: selectedTrace.executionId }) }}</p>
        <p class="trace-summary-primary" :data-tone="selectedTrace.summaryTone">{{ selectedTrace.summaryPrimary }}</p>
        <p v-if="selectedTrace.summarySecondary !== ''" class="trace-summary-secondary">{{ selectedTrace.summarySecondary }}</p>

        <div class="trace-steps">
          <div v-for="step in selectedTrace.steps" :key="step.id" class="trace-step" :data-tone="step.statusTone">
            <div class="trace-step-main">
              <span class="trace-step-title">{{ step.title }}</span>
              <span class="trace-step-summary">{{ step.summary }}</span>
              <span v-if="step.timestampLabel !== ''" class="trace-step-time">{{ step.timestampLabel }}</span>
            </div>
            <p v-if="step.detail !== ''" class="trace-step-detail">{{ step.detail }}</p>
            <details v-if="step.rawPayload !== ''" class="trace-step-raw">
              <summary class="trace-step-raw-summary">{{ t("conversation.trace.raw.expand") }}</summary>
              <pre class="trace-step-raw-content">{{ step.rawPayload }}</pre>
            </details>
          </div>
        </div>
      </template>
    </section>

    <section v-else-if="activeTab === 'files'" class="card">
      <strong>{{ t("conversation.inspector.tab.files") }}</strong>
      <p v-if="diff.length === 0" class="normal">{{ t("conversation.inspector.files.empty") }}</p>
      <ul v-else class="files-list">
        <li v-for="item in diff" :key="`${item.id}-file`">{{ item.path }}</li>
      </ul>
    </section>

    <section v-else class="card">
      <strong>{{ t("conversation.inspector.tab.risk") }}</strong>
      <p class="warning">{{ tf("conversation.inspector.risk.model", { model: modelLabel }) }}</p>
      <p class="normal">{{ riskSummary }}</p>
      <p class="normal">
        {{
          tf("conversation.inspector.risk.counts", {
            lowLabel: t("conversation.trace.risk.low"),
            low: riskLow,
            highLabel: t("conversation.trace.risk.high"),
            high: riskHigh,
            criticalLabel: t("conversation.trace.risk.critical"),
            critical: riskCritical
          })
        }}
      </p>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { computed, toRefs } from "vue";

import type { ExecutionTraceViewModel } from "@/modules/conversation/views/processTrace";
import { useI18n } from "@/shared/i18n";
import AppIcon from "@/shared/ui/AppIcon.vue";
import type { DiffCapability, DiffItem, Execution, ExecutionEvent, InspectorTabKey } from "@/shared/types/api";

defineEmits<{
  (event: "commit"): void;
  (event: "discard"): void;
  (event: "exportPatch"): void;
  (event: "changeTab", tab: InspectorTabKey): void;
  (event: "toggleCollapse"): void;
}>();

const props = withDefaults(defineProps<{
  diff: DiffItem[];
  capability: DiffCapability;
  queuedCount: number;
  pendingCount: number;
  executingCount: number;
  modelLabel: string;
  executions: Execution[];
  events: ExecutionEvent[];
  executionTraces?: ExecutionTraceViewModel[];
  selectedTraceExecutionId?: string;
  activeTab: InspectorTabKey;
}>(), {
  executionTraces: () => [],
  selectedTraceExecutionId: ""
});

const {
  activeTab,
  capability,
  diff,
  events,
  executions,
  executionTraces,
  executingCount,
  modelLabel,
  pendingCount,
  queuedCount,
  selectedTraceExecutionId
} = toRefs(props);

const { t } = useI18n();
const tabs = computed<Array<{ key: InspectorTabKey; label: string }>>(() => [
  { key: "diff", label: t("conversation.inspector.tab.diff") },
  { key: "run", label: t("conversation.inspector.tab.run") },
  { key: "trace", label: t("conversation.inspector.tab.trace") },
  { key: "files", label: t("conversation.inspector.tab.files") },
  { key: "risk", label: t("conversation.inspector.tab.risk") }
]);

const runHint = computed(() => {
  if (pendingCount.value > 0 || executingCount.value > 0) {
    return t("conversation.inspector.run.hint.executing");
  }
  if (queuedCount.value > 0) {
    return t("conversation.inspector.run.hint.queued");
  }
  return t("conversation.inspector.run.hint.idle");
});

const runHintTone = computed(() => (queuedCount.value > 0 ? "warning" : "normal"));

const latestExecution = computed(() =>
  [...executions.value].sort((left, right) => right.updated_at.localeCompare(left.updated_at))[0]
);

const latestExecutionLabel = computed(() => {
  if (!latestExecution.value) {
    return "";
  }
  return tf("conversation.inspector.run.latestExecution", {
    id: latestExecution.value.id,
    state: latestExecution.value.state
  });
});

const latestExecutionMetrics = computed(() => {
  const execution = latestExecution.value;
  if (!execution) {
    return "";
  }
  const startedAt = toDateOrNow(execution.created_at);
  const endedAt = isExecutionTerminal(execution.state) ? toDateOrNow(execution.updated_at) : new Date();
  const durationSec = Math.max(0, Math.round((endedAt.getTime() - startedAt.getTime()) / 1000));
  const tokensIn = toOptionalNonNegativeInteger(execution.tokens_in);
  const tokensOut = toOptionalNonNegativeInteger(execution.tokens_out);
  const tokenLabel = tokensIn === null || tokensOut === null
    ? t("conversation.inspector.run.tokenNotAvailable")
    : tf("conversation.inspector.run.tokenUsage", {
      input: tokensIn,
      output: tokensOut,
      total: tokensIn + tokensOut
    });
  return tf("conversation.inspector.run.metrics", { token: tokenLabel, duration: durationSec });
});

const selectedTrace = computed<ExecutionTraceViewModel | null>(() => {
  if (executionTraces.value.length <= 0) {
    return null;
  }
  const selectedExecutionId = selectedTraceExecutionId.value.trim();
  if (selectedExecutionId !== "") {
    const matched = executionTraces.value.find((trace) => trace.executionId === selectedExecutionId);
    if (matched) {
      return matched;
    }
  }
  return executionTraces.value[executionTraces.value.length - 1] ?? null;
});

const riskCounters = computed(() => {
  const counters = { low: 0, high: 0, critical: 0 };
  for (const event of events.value) {
    if (event.type !== "tool_call") {
      continue;
    }
    const riskLevel = typeof event.payload.risk_level === "string" ? event.payload.risk_level.trim().toLowerCase() : "";
    if (riskLevel === "critical") {
      counters.critical += 1;
      continue;
    }
    if (riskLevel === "high") {
      counters.high += 1;
      continue;
    }
    if (riskLevel === "low") {
      counters.low += 1;
    }
  }
  return counters;
});

const riskLow = computed(() => riskCounters.value.low);
const riskHigh = computed(() => riskCounters.value.high);
const riskCritical = computed(() => riskCounters.value.critical);
const riskSummary = computed(() => {
  const total = riskLow.value + riskHigh.value + riskCritical.value;
  if (total === 0) {
    return t("conversation.inspector.risk.summary.empty");
  }
  return tf("conversation.inspector.risk.summary.total", { total });
});

function mapChange(type: DiffItem["change_type"]): string {
  if (type === "added") {
    return "+";
  }
  if (type === "deleted") {
    return "-";
  }
  return "~";
}

function toDateOrNow(input: string): Date {
  const value = new Date(input);
  if (Number.isNaN(value.getTime())) {
    return new Date();
  }
  return value;
}

function isExecutionTerminal(state: Execution["state"]): boolean {
  return state === "completed" || state === "failed" || state === "cancelled";
}

function toOptionalNonNegativeInteger(value: unknown): number | null {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return null;
  }
  if (value < 0) {
    return 0;
  }
  return Math.trunc(value);
}

function tf(key: string, params: Record<string, string | number>): string {
  const template = t(key);
  return template.replace(/\{(\w+)\}/g, (_, token: string) => {
    if (!(token in params)) {
      return `{${token}}`;
    }
    return String(params[token]);
  });
}
</script>

<style scoped src="./MainInspectorPanel.css"></style>
