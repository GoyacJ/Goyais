<template>
  <WorkspaceSharedShell
    active-key="workspace_skills"
    title="技能配置"
    account-subtitle="Workspace Config / Skills"
    settings-subtitle="Local Settings / Skills"
  >
    <ResourceConfigTable
      title="技能列表"
      :columns="columns"
      :rows="listState.items as Array<Record<string, unknown>>"
      :empty-text="tableEmptyText"
      :search="listState.q"
      :loading="listState.loading"
      :can-prev="listState.page.backStack.length > 0"
      :can-next="listState.page.nextCursor !== null"
      :paging-loading="listState.page.loading"
      :add-disabled="!canWrite"
      search-placeholder="按名称搜索技能"
      add-label="新增技能"
      @update:search="onSearch"
      @add="openCreate"
      @prev="loadPreviousResourceConfigsPage('skill')"
      @next="loadNextResourceConfigsPage('skill')"
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
          <BaseButton :disabled="!canWrite" variant="ghost" @click="openEdit(row as ResourceConfig)">编辑</BaseButton>
          <BaseButton :disabled="!canWrite" variant="ghost" @click="toggleEnabled(row as ResourceConfig)">
            {{ (row as ResourceConfig).enabled ? "停用" : "启用" }}
          </BaseButton>
          <BaseButton :disabled="!canWrite" variant="ghost" @click="removeConfig(row as ResourceConfig)">删除</BaseButton>
        </div>
      </template>
    </ResourceConfigTable>

    <BaseModal :open="form.open" @close="closeModal">
      <template #title>
        <h3 class="modal-title">{{ form.mode === 'create' ? '新增技能' : '编辑技能' }}</h3>
      </template>

      <div class="modal-form">
        <label>
          名称
          <BaseInput v-model="form.name" placeholder="例如：代码评审技能" />
        </label>
        <MarkdownDualPaneEditor v-model="form.content" label="技能 Markdown" />
        <label class="switch">
          <input v-model="form.enabled" type="checkbox" />
          启用技能
        </label>
        <p v-if="form.message !== ''" class="modal-message">{{ form.message }}</p>
      </div>

      <template #footer>
        <div class="footer-actions">
          <BaseButton variant="ghost" @click="closeModal">取消</BaseButton>
          <BaseButton :disabled="!canWrite" variant="primary" @click="saveConfig">保存</BaseButton>
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
import BaseButton from "@/shared/ui/BaseButton.vue";
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
  tableEmptyText,
  loadNextResourceConfigsPage,
  loadPreviousResourceConfigsPage,
  onSearch,
  openCreate,
  openEdit,
  closeModal,
  removeConfig,
  saveConfig,
  toggleEnabled
} = useWorkspaceMarkdownResourceView("skill");
</script>

<style scoped src="./WorkspaceMarkdownView.css"></style>
