<template>
  <aside class="inspector">
    <p class="title">Inspector</p>
    <div class="tabs">
      <span class="tab active">Diff</span>
      <span class="tab">Run</span>
      <span class="tab">Files</span>
      <span class="tab">Risk</span>
    </div>

    <section class="card">
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

    <section class="card">
      <strong>Execution</strong>
      <p>Queue: {{ queuedCount }} · Active: {{ activeCount }}</p>
      <p class="warning">Approval: pending</p>
    </section>

    <section class="card">
      <strong>Resources / MCP</strong>
      <p>model: {{ modelId }} · mcp: 3 connected</p>
    </section>
  </aside>
</template>

<script setup lang="ts">
import type { DiffCapability, DiffItem } from "@/shared/types/api";

defineProps<{
  diff: DiffItem[];
  capability: DiffCapability;
  queuedCount: number;
  activeCount: number;
  modelId: string;
}>();

defineEmits<{
  (event: "commit"): void;
  (event: "discard"): void;
  (event: "exportPatch"): void;
}>();

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
  width: 340px;
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  padding: var(--global-space-12);
  display: grid;
  align-content: start;
  gap: var(--global-space-8);
}
.title {
  margin: 0;
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
  font-weight: var(--global-font-weight-600);
}
.tabs {
  display: inline-flex;
  gap: var(--global-space-8);
  font-size: var(--global-font-size-11);
}
.tab {
  color: var(--semantic-text-subtle);
}
.tab.active {
  color: var(--semantic-text);
  border-radius: var(--global-radius-8);
  background: var(--component-sidebar-item-bg-active);
  padding: var(--global-space-4) var(--global-space-8);
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
}
.action {
  border: 0;
  border-radius: var(--global-radius-8);
  background: var(--semantic-surface);
  color: var(--semantic-text);
  padding: var(--global-space-4) var(--global-space-8);
  font-size: var(--global-font-size-11);
}
.reason {
  margin: 0;
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
}
p {
  margin: 0;
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-11);
}
.warning {
  color: var(--semantic-warning);
}
</style>
