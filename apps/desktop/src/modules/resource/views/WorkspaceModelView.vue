<template>
  <WorkspaceSharedShell
    active-key="workspace_model"
    title="模型配置"
    account-subtitle="Workspace Config / Models"
    settings-subtitle="Local Settings / Models"
  >
    <p v-if="resourceStore.error" class="error">{{ resourceStore.error }}</p>

    <section class="meta-card">
      <div>
        <p class="meta-label">Catalog Root</p>
        <p class="meta-value">{{ resourceStore.catalogRoot || "-" }}</p>
      </div>
      <div>
        <p class="meta-label">Catalog Source</p>
        <p class="meta-value">{{ resourceStore.catalog?.source || "-" }}</p>
      </div>
      <div class="meta-actions">
        <button type="button" :disabled="resourceStore.catalogLoading" @click="reloadCatalog">手动刷新目录</button>
      </div>
    </section>

    <ResourceConfigTable
      title="模型列表"
      :columns="columns"
      :rows="resourceStore.models.items as Array<Record<string, unknown>>"
      :search="resourceStore.models.q"
      :loading="resourceStore.models.loading"
      :can-prev="resourceStore.models.page.backStack.length > 0"
      :can-next="resourceStore.models.page.nextCursor !== null"
      :paging-loading="resourceStore.models.page.loading"
      :add-disabled="!canWrite"
      search-placeholder="按名称搜索模型配置"
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
      <template #cell-model="{ row }">{{ (row as ResourceConfig).model?.model_id ?? "-" }}</template>
      <template #cell-probe="{ row }">
        <div class="probe-cell">
          <template v-if="getProbeResult((row as ResourceConfig).id)">
            <span :class="probeStatusClass(getProbeResult((row as ResourceConfig).id)?.status ?? '')">
              {{ getProbeResult((row as ResourceConfig).id)?.status === "success" ? "成功" : "失败" }}
            </span>
            <span class="probe-meta">
              延迟 {{ getProbeResult((row as ResourceConfig).id)?.latency_ms ?? 0 }}ms
              <template v-if="getProbeResult((row as ResourceConfig).id)?.error_code">
                / {{ getProbeResult((row as ResourceConfig).id)?.error_code }}
              </template>
            </span>
            <span class="probe-message">{{ getProbeResult((row as ResourceConfig).id)?.message }}</span>
            <span v-if="probeSuggestion(getProbeResult((row as ResourceConfig).id)?.error_code)" class="probe-suggest">
              建议：{{ probeSuggestion(getProbeResult((row as ResourceConfig).id)?.error_code) }}
            </span>
          </template>
          <span v-else class="probe-meta">未测试</span>
        </div>
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
          <button type="button" :disabled="!canWrite" @click="runModelTest(row as ResourceConfig)">测试</button>
          <button type="button" class="danger" :disabled="!canWrite" @click="removeConfig(row as ResourceConfig)">删除</button>
        </div>
      </template>
    </ResourceConfigTable>

    <BaseModal :open="form.open">
      <template #title>
        <h3 class="modal-title">{{ form.mode === 'create' ? '新增模型配置' : '编辑模型配置' }}</h3>
      </template>

      <div class="modal-grid">
        <label>
          名称
          <BaseInput v-model="form.name" placeholder="例如：OpenAI 主模型" />
        </label>
        <label>
          厂商
          <BaseSelect v-model="form.vendor" :options="vendorOptions" />
        </label>
        <label>
          模型（目录）
          <BaseSelect v-model="form.selectedCatalogModel" :options="vendorModelOptions" />
        </label>
        <label>
          模型 ID（可手输）
          <BaseInput v-model="form.modelId" placeholder="例如：gpt-4.1" />
        </label>
        <label>
          Base URL
          <BaseInput v-model="form.baseUrl" placeholder="https://api.openai.com/v1" />
        </label>
        <label>
          Timeout (ms)
          <BaseInput v-model="form.timeoutMs" placeholder="30000" />
        </label>
        <label class="full">
          API Key
          <input v-model="form.apiKey" class="native-input" type="password" placeholder="sk-..." />
          <span class="hint">{{ form.apiKeyHint }}</span>
        </label>
        <label class="full">
          Params(JSON)
          <textarea v-model="form.paramsText" class="native-textarea" placeholder='{"temperature":0.2}' />
        </label>
        <label class="switch">
          <input v-model="form.enabled" type="checkbox" />
          启用配置
        </label>
      </div>

      <template #footer>
        <button type="button" @click="closeModal">取消</button>
        <button type="button" :disabled="!canWrite" @click="saveConfig">保存</button>
      </template>
    </BaseModal>
  </WorkspaceSharedShell>
</template>

<script setup lang="ts">
import WorkspaceSharedShell from "@/shared/shells/WorkspaceSharedShell.vue";
import type { ResourceConfig } from "@/shared/types/api";
import BaseInput from "@/shared/ui/BaseInput.vue";
import BaseModal from "@/shared/ui/BaseModal.vue";
import BaseSelect from "@/shared/ui/BaseSelect.vue";
import ResourceConfigTable from "@/modules/resource/components/ResourceConfigTable.vue";
import { useWorkspaceModelView } from "@/modules/resource/views/useWorkspaceModelView";

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
  reloadCatalog,
  removeConfig,
  resourceStore,
  runModelTest,
  saveConfig,
  getProbeResult,
  probeStatusClass,
  probeSuggestion,
  toggleEnabled,
  vendorModelOptions,
  vendorOptions
} = useWorkspaceModelView();
</script>

<style scoped src="./WorkspaceModelView.css"></style>
