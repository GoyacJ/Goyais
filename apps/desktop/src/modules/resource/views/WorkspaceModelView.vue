<template>
  <WorkspaceSharedShell
    active-key="workspace_model"
    title="模型配置"
    account-subtitle="Workspace Config / Models"
    settings-subtitle="Local Settings / Models"
  >
    <section class="model-list-only">
      <ResourceConfigTable
        title="模型列表"
        :columns="columns"
        :rows="resourceStore.models.items as Array<Record<string, unknown>>"
        :empty-text="tableEmptyText"
        :search="resourceStore.models.q"
        :loading="resourceStore.models.loading"
        :can-prev="resourceStore.models.page.backStack.length > 0"
        :can-next="resourceStore.models.page.nextCursor !== null"
        :paging-loading="resourceStore.models.page.loading"
        :add-disabled="!canWrite"
        search-placeholder="按厂商或模型搜索"
        add-label="新增模型"
        @update:search="onSearch"
        @add="openCreate"
        @prev="loadPreviousResourceConfigsPage('model')"
        @next="loadNextResourceConfigsPage('model')"
      >
        <template #toolbar-right>
          <BaseSelect v-model="enabledFilterModel" :options="enabledOptions" :disabled="resourceStore.models.loading" />
        </template>

        <template #cell-vendor="{ row }">{{ (row as ResourceConfig).model?.vendor ?? "-" }}</template>
        <template #cell-model="{ row }">
          {{ (row as ResourceConfig).name?.trim() || (row as ResourceConfig).model?.model_id || "-" }}
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
            <BaseButton :disabled="!canWrite" variant="ghost" @click="runModelTest(row as ResourceConfig)">测试</BaseButton>
            <BaseButton :disabled="!canWrite" variant="ghost" @click="removeConfig(row as ResourceConfig)">删除</BaseButton>
          </div>
        </template>
      </ResourceConfigTable>
    </section>

    <BaseModal :open="form.open" @close="closeModal">
      <template #title>
        <h3 class="modal-title">{{ form.mode === "create" ? "新增模型配置" : "编辑模型配置" }}</h3>
      </template>

      <div class="modal-grid">
        <label>
          厂商
          <BaseSelect v-model="form.vendor" :options="vendorOptions" />
        </label>
        <label>
          模型
          <BaseSelect v-model="form.selectedCatalogModel" :options="vendorModelOptions" />
        </label>
        <label>
          模型名称（可选）
          <BaseInput v-model="form.name" placeholder="不填则默认使用模型 ID" />
        </label>
        <label v-if="showVendorEndpointSelector" class="full">
          Endpoint
          <BaseSelect v-model="form.baseUrlKey" :options="vendorEndpointOptions" />
          <span class="hint endpoint-hint">按目录 `base_url_key` 保存区域路由，不直接写入 `base_url`。</span>
        </label>
        <label v-if="showLocalBaseURL">
          Base URL
          <BaseInput v-model="form.baseUrl" :placeholder="selectedVendorBaseURL || 'http://127.0.0.1:11434/v1'" />
        </label>
        <label>
          Timeout (ms)
          <BaseInput v-model="form.timeoutMs" placeholder="30000" />
          <span class="hint">默认 30000，范围 1000-120000</span>
        </label>
        <label class="full">
          API Key
          <input v-model="form.apiKey" class="native-input" type="password" placeholder="sk-..." />
          <span class="hint">{{ form.apiKeyHint }}</span>
        </label>
        <label class="switch">
          <input v-model="form.enabled" type="checkbox" />
          启用配置
        </label>
      </div>

      <template #footer>
        <div class="modal-footer">
          <BaseButton variant="ghost" @click="closeModal">取消</BaseButton>
          <BaseButton :disabled="!canWrite" variant="primary" @click="saveConfig">保存</BaseButton>
        </div>
      </template>
    </BaseModal>

    <BaseModal :open="deleteConfirm.open" class="delete-modal" @close="closeDeleteConfirm">
      <template #title>
        <h3 class="modal-title">确认删除</h3>
      </template>
      <p class="delete-confirm-message">确认删除模型配置 {{ deleteConfirm.modelText }} ?</p>
      <template #footer>
        <div class="modal-footer">
          <BaseButton variant="ghost" @click="closeDeleteConfirm">取消</BaseButton>
          <BaseButton :disabled="!canWrite" variant="ghost" @click="confirmRemoveConfig">删除</BaseButton>
        </div>
      </template>
    </BaseModal>
  </WorkspaceSharedShell>
</template>

<script setup lang="ts">
import ResourceConfigTable from "@/modules/resource/components/ResourceConfigTable.vue";
import { useWorkspaceModelView } from "@/modules/resource/views/useWorkspaceModelView";
import WorkspaceSharedShell from "@/shared/shells/WorkspaceSharedShell.vue";
import type { ResourceConfig } from "@/shared/types/api";
import BaseButton from "@/shared/ui/BaseButton.vue";
import BaseInput from "@/shared/ui/BaseInput.vue";
import BaseModal from "@/shared/ui/BaseModal.vue";
import BaseSelect from "@/shared/ui/BaseSelect.vue";

const {
  canWrite,
  columns,
  enabledFilterModel,
  enabledOptions,
  form,
  formatTime,
  loadNextResourceConfigsPage,
  loadPreviousResourceConfigsPage,
  onSearch,
  openCreate,
  openEdit,
  closeModal,
  removeConfig,
  resourceStore,
  runModelTest,
  saveConfig,
  tableEmptyText,
  deleteConfirm,
  toggleEnabled,
  closeDeleteConfirm,
  confirmRemoveConfig,
  vendorModelOptions,
  vendorEndpointOptions,
  vendorOptions,
  showVendorEndpointSelector,
  showLocalBaseURL,
  selectedVendorBaseURL
} = useWorkspaceModelView();
</script>

<style scoped src="./WorkspaceModelView.css"></style>
