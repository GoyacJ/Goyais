<template>
  <aside class="inspector">
    <div class="head">
      <p class="title">Inspector</p>
      <button class="collapse-btn" type="button" title="最小化 Inspector" @click="$emit('toggleCollapse')">
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
        <strong>Diff</strong>
        <span>{{ diff.length }} files</span>
      </div>

      <div class="diff-list">
        <div v-for="item in diff" :key="item.id" class="diff-row">
          <span class="path">{{ item.path }}</span>
          <span class="stat" :class="item.change_type">{{ mapChange(item.change_type) }}</span>
        </div>
      </div>

      <div class="actions">
        <button class="action" type="button" :disabled="!capability.can_commit" @click="$emit('commit')">
          Commit
        </button>
        <button class="action" type="button" :disabled="!capability.can_discard" @click="$emit('discard')">
          Discard
        </button>
        <button class="action" type="button" :disabled="!capability.can_export_patch" @click="$emit('exportPatch')">
          Export Patch
        </button>
      </div>
      <p v-if="capability.reason" class="reason">{{ capability.reason }}</p>
    </section>

    <section v-else-if="activeTab === 'run'" class="card">
      <strong>Execution</strong>
      <p>Pending: {{ pendingCount }} · Executing: {{ executingCount }} · Queued: {{ queuedCount }}</p>
      <p v-if="latestExecutionLabel" class="normal">{{ latestExecutionLabel }}</p>
      <p v-if="latestExecutionMetrics" class="normal">{{ latestExecutionMetrics }}</p>
      <p :class="runHintTone">{{ runHint }}</p>
    </section>

    <section v-else-if="activeTab === 'files'" class="card">
      <strong>Files</strong>
      <p v-if="diff.length === 0" class="normal">暂无文件变更</p>
      <ul v-else class="files-list">
        <li v-for="item in diff" :key="`${item.id}-file`">{{ item.path }}</li>
      </ul>
    </section>

    <section v-else class="card">
      <strong>Risk</strong>
      <p class="warning">模型: {{ modelLabel }}</p>
      <p class="normal">{{ riskSummary }}</p>
      <p class="normal">low: {{ riskLow }} · high: {{ riskHigh }} · critical: {{ riskCritical }}</p>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { computed, toRefs } from "vue";

import AppIcon from "@/shared/ui/AppIcon.vue";
import type { DiffCapability, DiffItem, Execution, ExecutionEvent, InspectorTabKey } from "@/shared/types/api";

defineEmits<{
  (event: "commit"): void;
  (event: "discard"): void;
  (event: "exportPatch"): void;
  (event: "changeTab", tab: InspectorTabKey): void;
  (event: "toggleCollapse"): void;
}>();

const tabs: Array<{ key: InspectorTabKey; label: string }> = [
  { key: "diff", label: "Diff" },
  { key: "run", label: "Run" },
  { key: "files", label: "Files" },
  { key: "risk", label: "Risk" }
];

const props = defineProps<{
  diff: DiffItem[];
  capability: DiffCapability;
  queuedCount: number;
  pendingCount: number;
  executingCount: number;
  modelLabel: string;
  executions: Execution[];
  events: ExecutionEvent[];
  activeTab: InspectorTabKey;
}>();
const { activeTab, capability, diff, events, executions, executingCount, modelLabel, pendingCount, queuedCount } = toRefs(props);

const runHint = computed(() => {
  if (pendingCount.value > 0 || executingCount.value > 0) {
    return "执行中";
  }
  if (queuedCount.value > 0) {
    return "消息将按 FIFO 排队执行";
  }
  return "当前没有运行或排队任务";
});

const runHintTone = computed(() => (queuedCount.value > 0 ? "warning" : "normal"));

const latestExecution = computed(() =>
  [...executions.value].sort((left, right) => right.updated_at.localeCompare(left.updated_at))[0]
);

const latestExecutionLabel = computed(() => {
  if (!latestExecution.value) {
    return "";
  }
  return `最近执行: ${latestExecution.value.id} (${latestExecution.value.state})`;
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
    ? "Token N/A"
    : `Token in ${tokensIn} / out ${tokensOut} / total ${tokensIn + tokensOut}`;
  return `${tokenLabel} · 消息执行 ${durationSec}s`;
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
    return "当前会话暂无工具风险事件。";
  }
  return `当前会话累计 ${total} 次工具风险事件。`;
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
</script>

<style scoped src="./MainInspectorPanel.css"></style>
