<template>
  <WorkspaceSharedShell
    active-key="workspace_mcp"
    title="MCP 配置"
    account-subtitle="Workspace Config / MCP"
    settings-subtitle="Local Settings / MCP"
  >
    <p v-if="resourceStore.error" class="error">{{ resourceStore.error }}</p>

    <section class="toolbar">
      <BaseInput :model-value="listState.q" placeholder="按名称搜索 MCP" @update:model-value="onSearch" />
      <div class="toolbar-actions">
        <button type="button" :disabled="!canWrite" @click="openCreate">新增 MCP</button>
        <button type="button" @click="openExportModal">查看聚合 JSON</button>
      </div>
    </section>

    <section class="mcp-grid">
      <article v-for="item in listState.items" :key="item.id" class="mcp-card">
        <h3>{{ item.name }}</h3>
        <p class="meta">传输：{{ item.mcp?.transport ?? "-" }}</p>
        <p class="meta">Endpoint：{{ item.mcp?.endpoint || "-" }}</p>
        <p class="meta">Command：{{ item.mcp?.command || "-" }}</p>
        <p class="meta">工具数量：{{ item.mcp?.tools?.length ?? 0 }}</p>
        <p class="meta">最近连接：{{ formatTime(item.mcp?.last_connected_at) }}</p>
        <p class="meta" :class="item.enabled ? 'enabled' : 'disabled'">{{ item.enabled ? "启用" : "停用" }}</p>
        <p v-if="item.mcp?.last_error" class="meta disabled">错误：{{ item.mcp.last_error }}</p>
        <div class="diagnostic">
          <template v-if="getConnectResult(item.id)">
            <p class="meta" :class="connectStatusClass(getConnectResult(item.id)?.status ?? '')">
              最近探测：{{ getConnectResult(item.id)?.status === "connected" ? "连接成功" : "连接失败" }}
            </p>
            <p class="meta">工具拉取：{{ getConnectResult(item.id)?.tools.length ?? 0 }}</p>
            <p v-if="getConnectResult(item.id)?.error_code" class="meta disabled">
              错误码：{{ getConnectResult(item.id)?.error_code }}
            </p>
            <p class="meta">{{ getConnectResult(item.id)?.message }}</p>
            <p v-if="connectSuggestion(getConnectResult(item.id)?.error_code)" class="hint">
              建议：{{ connectSuggestion(getConnectResult(item.id)?.error_code) }}
            </p>
          </template>
          <p v-else class="meta">尚未发起连接探测</p>
        </div>

        <div class="card-actions">
          <button type="button" :disabled="!canWrite" @click="openEdit(item)">编辑</button>
          <button type="button" :disabled="!canWrite" @click="connect(item)">连接</button>
          <button type="button" :disabled="!canWrite" @click="toggleEnabled(item)">{{ item.enabled ? "停用" : "启用" }}</button>
          <button type="button" class="danger" :disabled="!canWrite" @click="removeConfig(item)">删除</button>
        </div>
      </article>

      <p v-if="listState.items.length === 0" class="empty">暂无 MCP 配置</p>
    </section>

    <BaseModal :open="form.open">
      <template #title>
        <h3 class="modal-title">{{ form.mode === 'create' ? '新增 MCP 配置' : '编辑 MCP 配置' }}</h3>
      </template>

      <div class="modal-form">
        <label>
          名称
          <BaseInput v-model="form.name" placeholder="例如：Workspace MCP" />
        </label>

        <label>
          传输方式
          <BaseSelect
            v-model="form.transport"
            :options="[
              { value: 'http_sse', label: 'http_sse' },
              { value: 'stdio', label: 'stdio' }
            ]"
          />
        </label>

        <label v-if="form.transport === 'http_sse'">
          Endpoint
          <BaseInput v-model="form.endpoint" placeholder="http://127.0.0.1:8000/sse" />
        </label>

        <label v-if="form.transport === 'stdio'">
          Command
          <BaseInput v-model="form.command" placeholder="npx @modelcontextprotocol/server-filesystem" />
        </label>

        <label>
          Env(JSON)
          <textarea v-model="form.envText" class="native-textarea" placeholder='{"TOKEN":"***"}' />
        </label>

        <label class="switch">
          <input v-model="form.enabled" type="checkbox" />
          启用配置
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

    <McpJsonModal :open="form.jsonModalOpen" :payload="resourceStore.mcpExport" @close="closeExportModal" />
  </WorkspaceSharedShell>
</template>

<script setup lang="ts">
import McpJsonModal from "@/modules/resource/components/McpJsonModal.vue";
import { useWorkspaceMcpView } from "@/modules/resource/views/useWorkspaceMcpView";
import WorkspaceSharedShell from "@/shared/shells/WorkspaceSharedShell.vue";
import BaseInput from "@/shared/ui/BaseInput.vue";
import BaseModal from "@/shared/ui/BaseModal.vue";
import BaseSelect from "@/shared/ui/BaseSelect.vue";

const {
  canWrite,
  closeExportModal,
  closeModal,
  connectStatusClass,
  connectSuggestion,
  connect,
  form,
  formatTime,
  getConnectResult,
  listState,
  onSearch,
  openCreate,
  openEdit,
  openExportModal,
  removeConfig,
  resourceStore,
  saveConfig,
  toggleEnabled
} = useWorkspaceMcpView();
</script>

<style scoped src="./WorkspaceMcpView.css"></style>
