<template>
  <WorkspaceSharedShell
    active-key="workspace_mcp"
    title="MCP 配置"
    account-subtitle="Workspace Config / MCP"
    settings-subtitle="Local Settings / MCP"
  >
    <section class="mcp-page">
      <p v-if="resourceStore.error" class="error">{{ resourceStore.error }}</p>

      <BaseCard class="toolbar-card">
        <div class="toolbar">
          <BaseInput :model-value="listState.q" placeholder="按名称搜索 MCP" @update:model-value="onSearch" />
          <div class="toolbar-actions">
            <BaseButton variant="secondary" :disabled="!canWrite" @click="openCreate">新增 MCP</BaseButton>
            <BaseButton variant="ghost" @click="openExportModal">MCP 配置</BaseButton>
          </div>
        </div>
      </BaseCard>

      <section class="mcp-grid" aria-label="MCP 配置列表">
        <BaseResourceCard v-for="item in listState.items" :key="item.id" details-label="连接详情">
          <template #title>
            <strong class="card-title">{{ item.name }}</strong>
          </template>

          <template #status>
            <StatusBadge :tone="item.enabled ? 'connected' : 'disconnected'" :label="item.enabled ? '启用' : '停用'" />
          </template>

          <p class="summary-row">
            <span>传输</span>
            <span>{{ item.mcp?.transport ?? "-" }}</span>
          </p>
          <p class="summary-row">
            <span>入口</span>
            <span class="mono">
              {{ item.mcp?.transport === "stdio" ? item.mcp?.command || "-" : item.mcp?.endpoint || "-" }}
            </span>
          </p>
          <p class="summary-row">
            <span>工具数量</span>
            <span>{{ item.mcp?.tools?.length ?? 0 }}</span>
          </p>
          <p class="summary-row">
            <span>最近连接</span>
            <span>{{ formatTime(item.mcp?.last_connected_at) }}</span>
          </p>
          <p class="summary-row">
            <span>最近探测</span>
            <span :class="getConnectResult(item.id) ? connectStatusClass(getConnectResult(item.id)?.status ?? '') : 'neutral'">
              {{
                getConnectResult(item.id)
                  ? getConnectResult(item.id)?.status === "connected"
                    ? "连接成功"
                    : "连接失败"
                  : "未探测"
              }}
            </span>
          </p>

          <template #details>
            <template v-if="getConnectResult(item.id)">
              <p class="detail-line">工具拉取：{{ getConnectResult(item.id)?.tools.length ?? 0 }}</p>
              <p class="detail-line">{{ getConnectResult(item.id)?.message }}</p>
              <p v-if="getConnectResult(item.id)?.error_code" class="detail-line disabled">
                错误码：{{ getConnectResult(item.id)?.error_code }}
              </p>
              <p v-if="connectSuggestion(getConnectResult(item.id)?.error_code)" class="detail-line hint">
                建议：{{ connectSuggestion(getConnectResult(item.id)?.error_code) }}
              </p>
            </template>
            <p v-else class="detail-line">尚未发起连接探测</p>
            <p v-if="item.mcp?.last_error" class="detail-line disabled">最近错误：{{ item.mcp.last_error }}</p>
          </template>

          <template #actionsPrimary>
            <div class="card-actions-row">
              <BaseButton variant="secondary" :disabled="!canWrite" @click="openEdit(item)">编辑</BaseButton>
              <BaseButton variant="secondary" :disabled="!canWrite" @click="connect(item)">连接</BaseButton>
              <BaseButton variant="secondary" :disabled="!canWrite" @click="toggleEnabled(item)">
                {{ item.enabled ? "停用" : "启用" }}
              </BaseButton>
              <BaseButton variant="danger" :disabled="!canWrite" @click="removeConfig(item)">删除</BaseButton>
            </div>
          </template>
        </BaseResourceCard>
      </section>

      <p v-if="listState.items.length === 0" class="empty">暂无 MCP 配置</p>

      <section class="pager-wrap">
        <CursorPager
          :can-prev="listState.page.backStack.length > 0"
          :can-next="listState.page.nextCursor !== null"
          :loading="listState.page.loading"
          @prev="loadPreviousPage"
          @next="loadNextPage"
        />
      </section>
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
          <BaseButton variant="ghost" @click="closeModal">取消</BaseButton>
          <BaseButton variant="primary" :disabled="!canWrite" @click="saveConfig">保存</BaseButton>
        </div>
      </template>
    </BaseModal>

    <McpJsonModal
      :open="form.jsonModalOpen"
      :payload="resourceStore.mcpExport"
      @close="closeExportModal"
      @save="applyExportPayload"
    />

  </WorkspaceSharedShell>
</template>

<script setup lang="ts">
import McpJsonModal from "@/modules/resource/components/McpJsonModal.vue";
import { useWorkspaceMcpView } from "@/modules/resource/views/useWorkspaceMcpView";
import WorkspaceSharedShell from "@/shared/shells/WorkspaceSharedShell.vue";
import BaseButton from "@/shared/ui/BaseButton.vue";
import BaseCard from "@/shared/ui/BaseCard.vue";
import BaseInput from "@/shared/ui/BaseInput.vue";
import BaseModal from "@/shared/ui/BaseModal.vue";
import BaseResourceCard from "@/shared/ui/BaseResourceCard.vue";
import BaseSelect from "@/shared/ui/BaseSelect.vue";
import CursorPager from "@/shared/ui/CursorPager.vue";
import StatusBadge from "@/shared/ui/StatusBadge.vue";

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
  loadNextPage,
  loadPreviousPage,
  onSearch,
  openCreate,
  openEdit,
  openExportModal,
  applyExportPayload,
  removeConfig,
  resourceStore,
  saveConfig,
  toggleEnabled
} = useWorkspaceMcpView();
</script>

<style scoped src="./WorkspaceMcpView.css"></style>
