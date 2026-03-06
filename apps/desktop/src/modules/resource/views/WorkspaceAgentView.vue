<template>
  <WorkspaceSharedShell
    active-key="workspace_agent"
    title="Agent 配置"
    account-subtitle="Workspace Config / Agent"
    settings-subtitle="Local Settings / Agent"
  >
    <section
      class="agent-config-card grid gap-[var(--global-space-12)] border border-[var(--semantic-border)] rounded-[var(--global-radius-12)] bg-[var(--semantic-surface)] p-[var(--global-space-16)]"
    >
      <header class="card-head">
        <h3 class="m-0 text-[var(--global-font-size-14)] text-[var(--semantic-text)]">执行配置</h3>
        <p class="mb-0 mt-[var(--global-space-4)] text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">
          配置变更仅对新建 execution 生效，运行中的 execution 不会切换。
        </p>
      </header>

      <div class="field-grid grid gap-[var(--global-space-12)]">
        <label class="field grid gap-[var(--global-space-6)]">
          <span class="field-label text-[var(--global-font-size-12)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
            Max Model Turns
          </span>
          <input
            class="field-input h-[32px] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] px-[var(--global-space-10)] text-[var(--global-font-size-12)] text-[var(--semantic-text)]"
            type="number"
            min="4"
            max="64"
            step="1"
            :value="config.execution.max_model_turns"
            :disabled="loading || saving || !canWrite"
            @change="onMaxTurnsChange"
          />
          <small class="field-hint text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">范围 4 - 64，默认 24</small>
        </label>

        <label class="field grid gap-[var(--global-space-6)]">
          <span class="field-label text-[var(--global-font-size-12)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
            过程展示
          </span>
          <BaseSelect
            v-model="traceToggleModel"
            :options="traceToggleOptions"
            :disabled="loading || saving || !canWrite"
          />
          <small class="field-hint text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">
            开启后在会话区展示 thinking/tool/command 过程轨迹
          </small>
        </label>

        <label class="field grid gap-[var(--global-space-6)]">
          <span class="field-label text-[var(--global-font-size-12)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
            展示粒度
          </span>
          <BaseSelect
            v-model="traceDetailModel"
            :options="traceDetailOptions"
            :disabled="loading || saving || !canWrite || !config.display.show_process_trace"
          />
          <small class="field-hint text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">
            `basic` 仅摘要，`verbose` 显示详细输出（截断）
          </small>
        </label>

        <label class="field grid gap-[var(--global-space-6)]">
          <span class="field-label text-[var(--global-font-size-12)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
            Default Mode
          </span>
          <BaseSelect
            v-model="defaultModeModel"
            :options="defaultModeOptions"
            :disabled="loading || saving || !canWrite"
          />
          <small class="field-hint text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">
            新 execution 默认采用的权限模式。
          </small>
        </label>

        <label class="field grid gap-[var(--global-space-6)]">
          <span class="field-label text-[var(--global-font-size-12)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
            Prompt Budget Chars
          </span>
          <input
            class="field-input h-[32px] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] px-[var(--global-space-10)] text-[var(--global-font-size-12)] text-[var(--semantic-text)]"
            type="number"
            min="256"
            max="512000"
            step="1"
            :value="config.capability_budgets.prompt_budget_chars"
            :disabled="loading || saving || !canWrite"
            @change="onPromptBudgetChange"
          />
          <small class="field-hint text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">
            控制能力 schema 注入预算，超阈值时会优先走 ToolSearch。
          </small>
        </label>

        <label class="field grid gap-[var(--global-space-6)]">
          <span class="field-label text-[var(--global-font-size-12)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
            Search Threshold %
          </span>
          <input
            class="field-input h-[32px] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] px-[var(--global-space-10)] text-[var(--global-font-size-12)] text-[var(--semantic-text)]"
            type="number"
            min="1"
            max="100"
            step="1"
            :value="config.capability_budgets.search_threshold_percent"
            :disabled="loading || saving || !canWrite"
            @change="onSearchThresholdChange"
          />
          <small class="field-hint text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">
            MCP schema 预算超过该比例时转为 searchable capability。
          </small>
        </label>

        <label class="field grid gap-[var(--global-space-6)]">
          <span class="field-label text-[var(--global-font-size-12)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
            MCP Search
          </span>
          <BaseSelect
            v-model="mcpSearchModel"
            :options="toggleOptions"
            :disabled="loading || saving || !canWrite"
          />
          <small class="field-hint text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">
            开启后允许通过 ToolSearch 发现延迟暴露的 MCP capability。
          </small>
        </label>

        <label class="field grid gap-[var(--global-space-6)]">
          <span class="field-label text-[var(--global-font-size-12)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
            MCP Result Limit
          </span>
          <input
            class="field-input h-[32px] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] px-[var(--global-space-10)] text-[var(--global-font-size-12)] text-[var(--semantic-text)]"
            type="number"
            min="1"
            max="200"
            step="1"
            :value="config.mcp_search.result_limit"
            :disabled="loading || saving || !canWrite || !config.mcp_search.enabled"
            @change="onMcpResultLimitChange"
          />
          <small class="field-hint text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">
            ToolSearch 返回的 capability 结果上限。
          </small>
        </label>

        <label class="field grid gap-[var(--global-space-6)]">
          <span class="field-label text-[var(--global-font-size-12)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
            Output Style
          </span>
          <input
            class="field-input h-[32px] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] px-[var(--global-space-10)] text-[var(--global-font-size-12)] text-[var(--semantic-text)]"
            type="text"
            :value="config.output_style"
            :disabled="loading || saving || !canWrite"
            @change="onOutputStyleChange"
          />
          <small class="field-hint text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">
            运行时默认输出风格标识，默认 `default`。
          </small>
        </label>

        <label class="field grid gap-[var(--global-space-6)]">
          <span class="field-label text-[var(--global-font-size-12)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
            Subagent Max Turns
          </span>
          <input
            class="field-input h-[32px] border border-[var(--semantic-border)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg)] px-[var(--global-space-10)] text-[var(--global-font-size-12)] text-[var(--semantic-text)]"
            type="number"
            min="1"
            max="64"
            step="1"
            :value="config.subagent_defaults.max_turns"
            :disabled="loading || saving || !canWrite"
            @change="onSubagentMaxTurnsChange"
          />
          <small class="field-hint text-[var(--global-font-size-11)] text-[var(--semantic-text-subtle)]">
            子 agent 的默认最大回合数。
          </small>
        </label>
      </div>

      <div class="summary-panel grid gap-[var(--global-space-6)] rounded-[var(--global-radius-8)] bg-[var(--semantic-bg-subtle)] p-[var(--global-space-12)]">
        <div class="text-[var(--global-font-size-12)] text-[var(--semantic-text)] [font-weight:var(--global-font-weight-600)]">
          Capability Graph Summary
        </div>
        <div class="text-[var(--global-font-size-11)] text-[var(--semantic-text-muted)]">
          Builtin tools: {{ config.builtin_tools.join(", ") || "none" }}
        </div>
        <div class="text-[var(--global-font-size-11)] text-[var(--semantic-text-muted)]">
          Feature flags: tool_search={{ config.feature_flags.enable_tool_search ? "on" : "off" }}, capability_graph={{ config.feature_flags.enable_capability_graph ? "on" : "off" }}
        </div>
      </div>

      <footer class="card-foot flex items-center justify-between gap-[var(--global-space-8)]">
        <span v-if="error !== ''" class="status text-[var(--global-font-size-12)] text-[var(--semantic-danger)]">{{ error }}</span>
        <span v-else-if="saving" class="status text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">保存中...</span>
        <span v-else-if="loading" class="status text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">加载中...</span>
        <span v-else class="status text-[var(--global-font-size-12)] text-[var(--semantic-text-muted)]">已就绪</span>
        <span v-if="!canWrite" class="readonly-tag text-[var(--global-font-size-11)] text-[var(--semantic-warning)]">只读权限</span>
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
import type { PermissionMode, TraceDetailLevel } from "@/shared/types/api";

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

const toggleOptions = [
  { value: "enabled", label: "开启" },
  { value: "disabled", label: "关闭" }
];

const defaultModeOptions = [
  { value: "default", label: "default" },
  { value: "acceptEdits", label: "acceptEdits" },
  { value: "plan", label: "plan" },
  { value: "dontAsk", label: "dontAsk" },
  { value: "bypassPermissions", label: "bypassPermissions" }
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

const defaultModeModel = computed({
  get: () => config.value.default_mode,
  set: (value: string) => {
    onDefaultModeChange(value);
  }
});

const mcpSearchModel = computed({
  get: () => (config.value.mcp_search.enabled ? "enabled" : "disabled"),
  set: (value: string) => {
    onMcpSearchToggleChange(value);
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

function onDefaultModeChange(value: string): void {
  void agentConfigStore.update({ default_mode: value as PermissionMode });
}

function onPromptBudgetChange(event: Event): void {
  const parsed = Number.parseInt((event.target as HTMLInputElement).value, 10);
  void agentConfigStore.update({ prompt_budget_chars: parsed });
}

function onSearchThresholdChange(event: Event): void {
  const parsed = Number.parseInt((event.target as HTMLInputElement).value, 10);
  void agentConfigStore.update({ search_threshold_percent: parsed });
}

function onMcpSearchToggleChange(value: string): void {
  void agentConfigStore.update({ mcp_search_enabled: value === "enabled" });
}

function onMcpResultLimitChange(event: Event): void {
  const parsed = Number.parseInt((event.target as HTMLInputElement).value, 10);
  void agentConfigStore.update({ mcp_search_result_limit: parsed });
}

function onOutputStyleChange(event: Event): void {
  void agentConfigStore.update({ output_style: (event.target as HTMLInputElement).value });
}

function onSubagentMaxTurnsChange(event: Event): void {
  const parsed = Number.parseInt((event.target as HTMLInputElement).value, 10);
  void agentConfigStore.update({ subagent_max_turns: parsed });
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
