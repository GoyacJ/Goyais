<template>
  <WorkspaceSharedShell
    active-key="workspace_agent"
    title="Agent 配置"
    account-subtitle="Workspace Config / Agent"
    settings-subtitle="Local Settings / Agent"
  >
    <section class="agent-config-card">
      <header class="card-head">
        <h3>执行配置</h3>
        <p>配置变更仅对新建 execution 生效，运行中的 execution 不会切换。</p>
      </header>

      <div class="field-grid">
        <label class="field">
          <span class="field-label">Max Model Turns</span>
          <input
            class="field-input"
            type="number"
            min="4"
            max="64"
            step="1"
            :value="config.execution.max_model_turns"
            :disabled="loading || saving || !canWrite"
            @change="onMaxTurnsChange"
          />
          <small class="field-hint">范围 4 - 64，默认 24</small>
        </label>

        <label class="field">
          <span class="field-label">过程展示</span>
          <BaseSelect
            v-model="traceToggleModel"
            :options="traceToggleOptions"
            :disabled="loading || saving || !canWrite"
          />
          <small class="field-hint">开启后在对话区展示 thinking/tool/command 过程轨迹</small>
        </label>

        <label class="field">
          <span class="field-label">展示粒度</span>
          <BaseSelect
            v-model="traceDetailModel"
            :options="traceDetailOptions"
            :disabled="loading || saving || !canWrite || !config.display.show_process_trace"
          />
          <small class="field-hint">`basic` 仅摘要，`verbose` 显示详细输出（截断）</small>
        </label>
      </div>

      <footer class="card-foot">
        <span v-if="error !== ''" class="status error">{{ error }}</span>
        <span v-else-if="saving" class="status">保存中...</span>
        <span v-else-if="loading" class="status">加载中...</span>
        <span v-else class="status">已就绪</span>
        <span v-if="!canWrite" class="readonly-tag">只读权限</span>
      </footer>
    </section>
  </WorkspaceSharedShell>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from "vue";

import { useWorkspaceAgentConfigStore } from "@/modules/resource/store/workspaceAgentConfigStore";
import { authStore } from "@/shared/stores/authStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import WorkspaceSharedShell from "@/shared/shells/WorkspaceSharedShell.vue";
import BaseSelect from "@/shared/ui/BaseSelect.vue";
import type { TraceDetailLevel } from "@/shared/types/api";

const agentConfigStore = useWorkspaceAgentConfigStore();
const config = computed(() => agentConfigStore.config.value);
const loading = computed(() => agentConfigStore.loading.value);
const saving = computed(() => agentConfigStore.saving.value);
const error = computed(() => agentConfigStore.error.value);
const canWrite = computed(() => workspaceStore.mode === "local" || authStore.capabilities.resource_write);

const traceToggleOptions = [
  { value: "enabled", label: "显示过程" },
  { value: "disabled", label: "隐藏过程" }
];

const traceDetailOptions = [
  { value: "verbose", label: "verbose" },
  { value: "basic", label: "basic" }
];

const traceToggleModel = computed({
  get: () => (config.value.display.show_process_trace ? "enabled" : "disabled"),
  set: (value: string) => {
    onTraceToggleChange(value);
  }
});

const traceDetailModel = computed({
  get: () => config.value.display.trace_detail_level,
  set: (value: string) => {
    onTraceDetailChange(value);
  }
});

function onMaxTurnsChange(event: Event): void {
  const raw = (event.target as HTMLInputElement).value;
  const parsed = Number.parseInt(raw, 10);
  void agentConfigStore.update({ max_model_turns: parsed });
}

function onTraceToggleChange(value: string): void {
  void agentConfigStore.update({ show_process_trace: value === "enabled" });
}

function onTraceDetailChange(value: string): void {
  const traceDetail = value === "basic" ? "basic" : "verbose";
  void agentConfigStore.update({ trace_detail_level: traceDetail as TraceDetailLevel });
}

onMounted(() => {
  void agentConfigStore.load(true);
});

watch(
  () => workspaceStore.currentWorkspaceId,
  () => {
    void agentConfigStore.load(true);
  }
);
</script>

<style scoped>
.agent-config-card {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  padding: var(--global-space-16);
  display: grid;
  gap: var(--global-space-12);
}

.card-head h3 {
  margin: 0;
  color: var(--semantic-text);
  font-size: var(--global-font-size-14);
}

.card-head p {
  margin: var(--global-space-4) 0 0;
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
}

.field-grid {
  display: grid;
  gap: var(--global-space-12);
}

.field {
  display: grid;
  gap: var(--global-space-6);
}

.field-label {
  color: var(--semantic-text);
  font-size: var(--global-font-size-12);
  font-weight: var(--global-font-weight-600);
}

.field-input {
  border: 1px solid var(--semantic-border);
  border-radius: var(--global-radius-8);
  background: var(--semantic-bg);
  color: var(--semantic-text);
  height: 32px;
  padding: 0 var(--global-space-10);
  font-size: var(--global-font-size-12);
}

.field-hint {
  color: var(--semantic-text-subtle);
  font-size: var(--global-font-size-11);
}

.card-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--global-space-8);
}

.status {
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
}

.status.error {
  color: var(--semantic-danger);
}

.readonly-tag {
  color: var(--semantic-warning);
  font-size: var(--global-font-size-11);
}
</style>
