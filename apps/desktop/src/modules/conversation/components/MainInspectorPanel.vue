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
        <span>{{ tf("conversation.inspector.diff.filesCount", { count: diffEntries.length }) }}</span>
      </div>

      <p v-if="diffEntries.length === 0" class="normal">{{ t("conversation.inspector.diff.empty") }}</p>
      <div v-else class="diff-list">
        <div v-for="item in diffEntries" :key="item.entry_id" class="diff-row">
          <span class="path">{{ item.path }}</span>
          <span v-if="isDiffLineCountUnknown(item)" class="stat stat-unknown">--</span>
          <span v-else class="stat">
            <span class="stat-added">+{{ displayDiffLineCount(item.added_lines) }}</span>
            <span class="stat-separator"> / </span>
            <span class="stat-deleted">-{{ displayDiffLineCount(item.deleted_lines) }}</span>
          </span>
        </div>
      </div>

      <div class="actions">
        <button class="action" type="button" :disabled="!capability.can_commit || diffEntries.length === 0" @click="$emit('commit')">
          {{ t("conversation.inspector.action.commit") }}
        </button>
        <button class="action" type="button" :disabled="!capability.can_discard || diffEntries.length === 0" @click="$emit('discard')">
          {{ t("conversation.inspector.action.discard") }}
        </button>
        <button class="action" type="button" :disabled="!canExport" @click="$emit('exportPatch')">
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
      <p v-if="graphRunTaskItems.length > 0" class="normal">
        Tasks: {{ graphRunTaskItems.length }} · running {{ runTaskStateCounts.running }} · queued {{ runTaskStateCounts.queued }} · blocked {{ runTaskStateCounts.blocked }}
      </p>
      <p v-if="runTaskGraphLoading" class="normal">Loading task graph...</p>
      <div class="actions">
        <button class="action" type="button" @click="$emit('refreshRunTasks')">Refresh tasks</button>
        <select class="action" :value="runTaskStateFilter" @change="onRunTaskStateFilterChange">
          <option value="">All tasks</option>
          <option value="queued">Queued</option>
          <option value="blocked">Blocked</option>
          <option value="running">Running</option>
          <option value="retrying">Retrying</option>
          <option value="completed">Completed</option>
          <option value="failed">Failed</option>
          <option value="cancelled">Cancelled</option>
        </select>
      </div>
      <p v-if="runTaskListLoading" class="normal">Loading task list...</p>
      <p v-if="!runTaskListLoading && displayRunTaskItems.length === 0" class="normal">No task graph data.</p>
      <div v-else-if="displayRunTaskItems.length > 0" class="diff-list">
        <div v-for="task in displayRunTaskItems" :key="task.task_id" class="diff-row">
          <button class="action" type="button" @click="$emit('selectRunTask', task.task_id)">
            {{ task.title || task.task_id }}
          </button>
          <div class="actions">
            <span class="stat">{{ task.state }}</span>
            <button
              v-for="action in resolveRunTaskControlActions(task)"
              :key="`${task.task_id}:${action}`"
              class="action"
              type="button"
              :disabled="runTaskGraphLoading"
              @click.stop="$emit('controlRunTask', { taskId: task.task_id, action })"
            >
              {{ resolveRunTaskControlLabel(action) }}
            </button>
          </div>
        </div>
      </div>
      <div v-if="runTaskListNextCursor !== null" class="actions">
        <button class="action" type="button" :disabled="runTaskListLoading" @click="$emit('loadMoreRunTasks')">Load more tasks</button>
      </div>
      <p v-if="runTaskDetailLoading" class="normal">Loading task detail...</p>
      <template v-else-if="selectedRunTask">
        <p class="normal">Task: {{ selectedRunTask.title || selectedRunTask.task_id }}</p>
        <p class="normal">Task ID: {{ selectedRunTask.task_id }}</p>
        <p class="normal">State: {{ selectedRunTask.state }}</p>
        <p class="normal">
          Depends on: {{ selectedRunTask.depends_on.length > 0 ? selectedRunTask.depends_on.join(", ") : "None" }}
        </p>
        <p class="normal">Retry: {{ selectedRunTask.retry_count }} / {{ selectedRunTask.max_retries }}</p>
        <p v-if="selectedRunTask.last_error" class="warning">{{ selectedRunTask.last_error }}</p>
        <template v-if="selectedRunTask.artifact">
          <p class="normal">Artifact: {{ selectedRunTask.artifact.kind }}</p>
          <p v-if="selectedRunTask.artifact.summary" class="normal">{{ selectedRunTask.artifact.summary }}</p>
          <p v-if="selectedRunTask.artifact.uri" class="normal">{{ selectedRunTask.artifact.uri }}</p>
        </template>
      </template>
      <p :class="runHintTone">{{ runHint }}</p>
    </section>

    <section v-else-if="activeTab === 'trace'" class="card trace-card">
      <strong>{{ t("conversation.inspector.tab.trace") }}</strong>
      <p v-if="traceMessageItems.length === 0" class="normal">{{ t("conversation.inspector.trace.empty") }}</p>
      <template v-else>
        <div class="trace-message-list">
          <button
            v-for="item in traceMessageItems"
            :key="item.id"
            class="trace-message-btn"
            :class="{ active: item.id === selectedTraceMessage?.id }"
            type="button"
            @click="$emit('selectTraceMessage', item.id)"
          >
            <span class="trace-message-text">{{ item.preview }}</span>
            <span class="trace-message-meta">{{ tf("conversation.inspector.trace.executionCount", { count: item.traces.length }) }}</span>
          </button>
        </div>

        <p v-if="selectedTraceMessageTraces.length === 0" class="normal">{{ t("conversation.inspector.trace.messageEmpty") }}</p>
        <template v-else>
          <div class="trace-execution-list">
            <button
              v-for="trace in selectedTraceMessageTraces"
              :key="trace.executionId"
              class="trace-execution-btn"
              :class="{ active: trace.executionId === selectedTrace?.executionId }"
              type="button"
              @click="$emit('selectTraceExecution', trace.executionId)"
            >
              {{ tf("conversation.inspector.trace.executionShort", { id: trace.executionId }) }}
            </button>
          </div>

          <template v-if="selectedTrace">
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
        </template>
      </template>
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
import type {
  ChangeEntry,
  SessionChangeSet,
  SessionMessage,
  DiffItem,
  Run,
  RunLifecycleEvent,
  InspectorTabKey,
  OpenAPIContractComponents
} from "@/shared/types/api";

type InspectorCapability = {
  can_commit: boolean;
  can_discard: boolean;
  can_export?: boolean;
  can_export_patch?: boolean;
  reason?: string;
};

type RunTaskNode = OpenAPIContractComponents["schemas"]["TaskNode"];
type RunTaskGraph = OpenAPIContractComponents["schemas"]["AgentGraph"];
type RunTaskState = OpenAPIContractComponents["schemas"]["TaskState"];
type RunTaskControlAction = OpenAPIContractComponents["schemas"]["TaskControlRequest"]["action"];

const emit = defineEmits<{
  (event: "commit"): void;
  (event: "discard"): void;
  (event: "exportPatch"): void;
  (event: "refreshRunTasks"): void;
  (event: "changeRunTaskStateFilter", state: RunTaskState | ""): void;
  (event: "selectRunTask", taskId: string): void;
  (event: "loadMoreRunTasks"): void;
  (event: "controlRunTask", input: { taskId: string; action: RunTaskControlAction }): void;
  (event: "changeTab", tab: InspectorTabKey): void;
  (event: "selectTraceMessage", messageId: string): void;
  (event: "selectTraceExecution", executionId: string): void;
  (event: "toggleCollapse"): void;
}>();

const props = withDefaults(defineProps<{
  changeSet?: SessionChangeSet | null;
  diff?: DiffItem[];
  capability: InspectorCapability;
  queuedCount: number;
  pendingCount: number;
  executingCount: number;
  modelLabel: string;
  runTaskGraph?: RunTaskGraph | null;
  runTaskGraphLoading?: boolean;
  runTaskItems?: RunTaskNode[];
  runTaskListLoading?: boolean;
  runTaskListNextCursor?: string | null;
  runTaskStateFilter?: RunTaskState | "";
  selectedRunTask?: RunTaskNode | null;
  runTaskDetailLoading?: boolean;
  executions: Run[];
  events: RunLifecycleEvent[];
  messages?: SessionMessage[];
  executionTraces?: ExecutionTraceViewModel[];
  selectedTraceMessageId?: string;
  selectedTraceExecutionId?: string;
  activeTab: InspectorTabKey;
}>(), {
  changeSet: null,
  diff: () => [],
  messages: () => [],
  executionTraces: () => [],
  selectedTraceMessageId: "",
  selectedTraceExecutionId: "",
  runTaskGraph: null,
  runTaskGraphLoading: false,
  runTaskItems: () => [],
  runTaskListLoading: false,
  runTaskListNextCursor: null,
  runTaskStateFilter: "",
  selectedRunTask: null,
  runTaskDetailLoading: false
});

const {
  activeTab,
  capability,
  changeSet,
  diff,
  events,
  executions,
  messages,
  executionTraces,
  executingCount,
  modelLabel,
  runTaskGraph,
  runTaskGraphLoading,
  runTaskItems,
  runTaskListLoading,
  runTaskListNextCursor,
  runTaskStateFilter,
  selectedRunTask,
  runTaskDetailLoading,
  pendingCount,
  queuedCount,
  selectedTraceMessageId,
  selectedTraceExecutionId
} = toRefs(props);

const { t } = useI18n();
const tabs = computed<Array<{ key: InspectorTabKey; label: string }>>(() => [
  { key: "diff", label: t("conversation.inspector.tab.diff") },
  { key: "run", label: t("conversation.inspector.tab.run") },
  { key: "trace", label: t("conversation.inspector.tab.trace") },
  { key: "risk", label: t("conversation.inspector.tab.risk") }
]);
const canExport = computed(() => capability.value.can_export ?? capability.value.can_export_patch ?? true);
const diffEntries = computed<ChangeEntry[]>(() => {
  if (changeSet.value?.entries && changeSet.value.entries.length > 0) {
    return changeSet.value.entries;
  }
  return (diff.value ?? []).map((item) => ({
    entry_id: item.id,
    message_id: "",
    execution_id: "",
    path: item.path,
    change_type: item.change_type,
    summary: item.summary,
    added_lines: item.added_lines,
    deleted_lines: item.deleted_lines,
    created_at: ""
  }));
});

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

const graphRunTaskItems = computed<RunTaskNode[]>(() => runTaskGraph.value?.tasks ?? []);
const displayRunTaskItems = computed<RunTaskNode[]>(() =>
  runTaskItems.value.length > 0 ? runTaskItems.value : graphRunTaskItems.value
);
const runTaskStateCounts = computed(() => {
  return graphRunTaskItems.value.reduce(
    (acc, task) => {
      const normalized = task.state.trim().toLowerCase();
      if (normalized === "running") {
        acc.running += 1;
      } else if (normalized === "queued") {
        acc.queued += 1;
      } else if (normalized === "blocked") {
        acc.blocked += 1;
      }
      return acc;
    },
    { running: 0, queued: 0, blocked: 0 }
  );
});

function onRunTaskStateFilterChange(event: Event): void {
  const nextState = ((event.target as HTMLSelectElement | null)?.value ?? "").trim();
  const normalized = normalizeRunTaskState(nextState);
  if (normalized === null) {
    return;
  }
  emit("changeRunTaskStateFilter", normalized);
}

function normalizeRunTaskState(input: string): RunTaskState | "" | null {
  if (input === "") {
    return "";
  }
  if (
    input === "queued" ||
    input === "blocked" ||
    input === "running" ||
    input === "retrying" ||
    input === "completed" ||
    input === "failed" ||
    input === "cancelled"
  ) {
    return input;
  }
  return null;
}

function resolveRunTaskControlActions(task: RunTaskNode): RunTaskControlAction[] {
  const state = task.state.trim().toLowerCase();
  if (state === "queued" || state === "blocked" || state === "running" || state === "retrying") {
    return ["cancel"];
  }
  return [];
}

function resolveRunTaskControlLabel(action: RunTaskControlAction): string {
  switch (action) {
    case "cancel":
      return "Cancel";
    case "retry":
      return "Retry";
    case "pause":
      return "Pause";
    case "resume":
      return "Resume";
    default:
      return action;
  }
}

const traceMessageItems = computed<Array<{
  id: string;
  preview: string;
  queueIndex: number;
  traces: ExecutionTraceViewModel[];
}>>(() => {
  const tracesByMessageId = new Map<string, ExecutionTraceViewModel[]>();
  const tracesByQueueIndex = new Map<number, ExecutionTraceViewModel[]>();
  for (const trace of executionTraces.value) {
    const messageId = trace.messageId.trim();
    if (messageId !== "") {
      const list = tracesByMessageId.get(messageId) ?? [];
      list.push(trace);
      tracesByMessageId.set(messageId, list);
    }
    const queueIndex = trace.queueIndex;
    const queueList = tracesByQueueIndex.get(queueIndex) ?? [];
    queueList.push(trace);
    tracesByQueueIndex.set(queueIndex, queueList);
  }

  const result: Array<{
    id: string;
    preview: string;
    queueIndex: number;
    traces: ExecutionTraceViewModel[];
  }> = [];
  for (const message of messages.value) {
    if (message.role !== "user") {
      continue;
    }
    const messageId = message.id.trim();
    if (messageId === "") {
      continue;
    }
    const directMatches = tracesByMessageId.get(messageId) ?? [];
    const queueMatches = typeof message.queue_index === "number"
      ? tracesByQueueIndex.get(message.queue_index) ?? []
      : [];
    const merged = new Map<string, ExecutionTraceViewModel>();
    for (const trace of directMatches) {
      merged.set(trace.executionId, trace);
    }
    for (const trace of queueMatches) {
      merged.set(trace.executionId, trace);
    }
    result.push({
      id: messageId,
      preview: buildTraceMessagePreview(message.content),
      queueIndex: typeof message.queue_index === "number" ? message.queue_index : Number.MAX_SAFE_INTEGER,
      traces: [...merged.values()].sort((left, right) => left.queueIndex - right.queueIndex)
    });
  }
  return result;
});

const selectedTraceMessage = computed(() => {
  if (traceMessageItems.value.length === 0) {
    return null;
  }
  const selectedMessageId = selectedTraceMessageId.value.trim();
  if (selectedMessageId !== "") {
    const matched = traceMessageItems.value.find((message) => message.id === selectedMessageId);
    if (matched) {
      return matched;
    }
  }
  const latestWithTrace = [...traceMessageItems.value].reverse().find((item) => item.traces.length > 0);
  return latestWithTrace ?? traceMessageItems.value[traceMessageItems.value.length - 1] ?? null;
});

const selectedTraceMessageTraces = computed(() => selectedTraceMessage.value?.traces ?? []);

const selectedTrace = computed<ExecutionTraceViewModel | null>(() => {
  const traces = selectedTraceMessageTraces.value;
  if (traces.length <= 0) {
    return null;
  }
  const selectedExecutionId = selectedTraceExecutionId.value.trim();
  if (selectedExecutionId !== "") {
    const matched = traces.find((trace) => trace.executionId === selectedExecutionId);
    if (matched) {
      return matched;
    }
  }
  return traces[traces.length - 1] ?? null;
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

function isDiffLineCountUnknown(item: ChangeEntry): boolean {
  const added = toOptionalNonNegativeInteger(item.added_lines);
  const deleted = toOptionalNonNegativeInteger(item.deleted_lines);
  return added === null && deleted === null;
}

function displayDiffLineCount(value: unknown): number {
  return toOptionalNonNegativeInteger(value) ?? 0;
}

function buildTraceMessagePreview(content: string): string {
  const normalized = content.trim().replace(/\s+/g, " ");
  if (normalized === "") {
    return t("conversation.inspector.trace.messageFallback");
  }
  if (normalized.length <= 60) {
    return normalized;
  }
  return `${normalized.slice(0, 57)}...`;
}

function toDateOrNow(input: string): Date {
  const value = new Date(input);
  if (Number.isNaN(value.getTime())) {
    return new Date();
  }
  return value;
}

function isExecutionTerminal(state: Run["state"]): boolean {
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
