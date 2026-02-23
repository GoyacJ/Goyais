<template>
  <WorkspaceSharedShell
    active-key="workspace_rules"
    title="规则配置"
    account-subtitle="Workspace Config / Rules"
    settings-subtitle="Local Settings / Rules"
  >
    <p v-if="resourceStore.error" class="error">{{ resourceStore.error }}</p>

    <ResourceConfigTable
      title="规则列表"
      :columns="columns"
      :rows="listState.items as Array<Record<string, unknown>>"
      :search="listState.q"
      :loading="listState.loading"
      :can-prev="listState.page.backStack.length > 0"
      :can-next="listState.page.nextCursor !== null"
      :paging-loading="listState.page.loading"
      :add-disabled="!canWrite"
      search-placeholder="按名称搜索规则"
      add-label="新增规则"
      @update:search="onSearch"
      @add="openCreate"
      @prev="loadPreviousResourceConfigsPage('rule')"
      @next="loadNextResourceConfigsPage('rule')"
    >
      <template #toolbar-right>
        <BaseSelect v-model="enabledFilterModel" :options="enabledOptions" :disabled="listState.loading" />
      </template>

      <template #cell-enabled="{ row }">
        <span :class="(row as ResourceConfig).enabled ? 'enabled' : 'disabled'">
          {{ (row as ResourceConfig).enabled ? "启用" : "停用" }}
        </span>
      </template>
      <template #cell-updated="{ row }">{{ formatTime((row as ResourceConfig).updated_at) }}</template>
      <template #cell-actions="{ row }">
        <div class="actions">
          <button type="button" :disabled="!canWrite" @click="openEdit(row as ResourceConfig)">编辑</button>
          <button type="button" :disabled="!canWrite" @click="toggleEnabled(row as ResourceConfig)">
            {{ (row as ResourceConfig).enabled ? "停用" : "启用" }}
          </button>
          <button type="button" class="danger" :disabled="!canWrite" @click="removeConfig(row as ResourceConfig)">删除</button>
        </div>
      </template>
    </ResourceConfigTable>

    <BaseModal :open="form.open">
      <template #title>
        <h3 class="modal-title">{{ form.mode === 'create' ? '新增规则' : '编辑规则' }}</h3>
      </template>

      <div class="modal-form">
        <label>
          名称
          <BaseInput v-model="form.name" placeholder="例如：仓库安全规则" />
        </label>
        <MarkdownDualPaneEditor v-model="form.content" label="规则 Markdown" />
        <label class="switch">
          <input v-model="form.enabled" type="checkbox" />
          启用规则
        </label>
        <p v-if="form.message !== ''" class="modal-message">{{ form.message }}</p>
      </div>

      <template #footer>
        <div class="footer-actions">
          <button type="button" @click="closeModal">取消</button>
          <button type="button" :disabled="!canWrite" @click="saveConfig">保存</button>
        </div>
      </template>
    </BaseModal>
  </WorkspaceSharedShell>
</template>

<script setup lang="ts">
import MarkdownDualPaneEditor from "@/modules/resource/components/MarkdownDualPaneEditor.vue";
import ResourceConfigTable from "@/modules/resource/components/ResourceConfigTable.vue";
import { useWorkspaceMarkdownResourceView } from "@/modules/resource/views/useWorkspaceMarkdownResourceView";
import WorkspaceSharedShell from "@/shared/shells/WorkspaceSharedShell.vue";
import type { ResourceConfig } from "@/shared/types/api";
import BaseInput from "@/shared/ui/BaseInput.vue";
import BaseModal from "@/shared/ui/BaseModal.vue";
import BaseSelect from "@/shared/ui/BaseSelect.vue";

const columns = [
  { key: "name", label: "名称" },
  { key: "enabled", label: "状态" },
  { key: "updated", label: "更新时间" },
  { key: "actions", label: "动作" }
];

const {
  canWrite,
  enabledFilterModel,
  enabledOptions,
  form,
  formatTime,
  listState,
  loadNextResourceConfigsPage,
  loadPreviousResourceConfigsPage,
  onSearch,
  openCreate,
  openEdit,
  closeModal,
  removeConfig,
  resourceStore,
  saveConfig,
  toggleEnabled
} = useWorkspaceMarkdownResourceView("rule");
</script>

<style scoped src="./WorkspaceMarkdownView.css"></style>
