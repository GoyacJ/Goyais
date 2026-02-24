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
      <p>Pending: {{ pendingCount }} · Executing: {{ executingCount }} · Confirming: {{ confirmingCount }} · Queued: {{ queuedCount }}</p>
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
      <p class="warning">模型: {{ modelId }}</p>
      <p class="normal">高风险操作需审批并写审计。</p>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { computed, toRefs } from "vue";

import AppIcon from "@/shared/ui/AppIcon.vue";
import type { DiffCapability, DiffItem, InspectorTabKey } from "@/shared/types/api";

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
  confirmingCount: number;
  modelId: string;
  activeTab: InspectorTabKey;
}>();
const { activeTab, capability, confirmingCount, diff, executingCount, modelId, pendingCount, queuedCount } = toRefs(props);

const runHint = computed(() => {
  if (confirmingCount.value > 0) {
    return "执行正在等待风险确认";
  }
  if (pendingCount.value > 0 || executingCount.value > 0) {
    return "执行中";
  }
  if (queuedCount.value > 0) {
    return "消息将按 FIFO 排队执行";
  }
  return "当前没有运行或排队任务";
});

const runHintTone = computed(() => (confirmingCount.value > 0 || queuedCount.value > 0 ? "warning" : "normal"));

function mapChange(type: DiffItem["change_type"]): string {
  if (type === "added") {
    return "+";
  }
  if (type === "deleted") {
    return "-";
  }
  return "~";
}
</script>

<style scoped>
.inspector {
  width: 280px;
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  padding: var(--global-space-12);
  display: grid;
  align-content: start;
  gap: var(--global-space-8);
}

.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--global-space-8);
}

.title {
  margin: 0;
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
  font-weight: var(--global-font-weight-600);
}

.collapse-btn {
  width: 22px;
  height: 22px;
  border-radius: var(--global-radius-8);
  border: 0;
  background: transparent;
  color: var(--semantic-text-subtle);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.collapse-btn:hover {
  background: var(--semantic-surface-2);
  color: var(--semantic-text);
}

.tabs {
  display: inline-flex;
  gap: var(--global-space-4);
}

.tab {
  border: 0;
  border-radius: var(--global-radius-8);
  background: transparent;
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
  padding: var(--global-space-4) var(--global-space-8);
}

.tab.active {
  color: var(--semantic-text);
  background: var(--component-sidebar-item-bg-active);
}

.card {
  background: var(--semantic-bg);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8);
  display: grid;
  gap: var(--global-space-8);
}

.card-head {
  display: flex;
  justify-content: space-between;
}

.diff-list {
  display: grid;
  gap: var(--global-space-4);
}

.diff-row {
  background: var(--semantic-surface);
  border-radius: var(--global-radius-8);
  padding: var(--global-space-8);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--global-space-8);
}

.path {
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-11);
  word-break: break-word;
}

.stat {
  font-size: var(--global-font-size-12);
  font-weight: var(--global-font-weight-700);
}

.stat.added {
  color: var(--semantic-success);
}

.stat.deleted {
  color: var(--semantic-danger);
}

.stat.modified {
  color: var(--semantic-warning);
}

.actions {
  display: flex;
  gap: var(--global-space-8);
  flex-wrap: wrap;
}

.action {
  border: 0;
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface);
  color: var(--semantic-text);
  padding: var(--global-space-4) var(--global-space-8);
  font-size: var(--global-font-size-11);
}

.reason,
.normal,
.warning,
p {
  margin: 0;
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-11);
}

.warning {
  color: var(--semantic-warning);
}

.files-list {
  margin: 0;
  padding-left: var(--global-space-16);
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-11);
  display: grid;
  gap: var(--global-space-4);
}
</style>
