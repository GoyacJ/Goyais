import { computed, reactive, watch } from "vue";

import {
  connectWorkspaceMcpConfig,
  createWorkspaceResourceConfig,
  deleteWorkspaceResourceConfig,
  patchWorkspaceResourceConfig,
  refreshResourceConfigsByType,
  refreshWorkspaceMcpExport,
  resourceStore,
  setResourceSearch
} from "@/modules/resource/store";
import { authStore } from "@/shared/stores/authStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import type { ResourceConfig } from "@/shared/types/api";

export function useWorkspaceMcpView() {
  const canWrite = computed(() => workspaceStore.mode === "local" || authStore.capabilities.resource_write);
  const listState = computed(() => resourceStore.mcps);

  const form = reactive({
    open: false,
    mode: "create" as "create" | "edit",
    configId: "",
    name: "",
    transport: "http_sse" as "http_sse" | "stdio",
    endpoint: "",
    command: "",
    envText: "",
    enabled: true,
    message: "",
    jsonModalOpen: false
  });

  watch(
    () => workspaceStore.currentWorkspaceId,
    async () => {
      await refreshResourceConfigsByType("mcp");
    },
    { immediate: true }
  );

  function onSearch(value: string): void {
    setResourceSearch("mcp", value);
    void refreshResourceConfigsByType("mcp");
  }

  function openCreate(): void {
    if (!canWrite.value) return;
    Object.assign(form, {
      open: true,
      mode: "create",
      configId: "",
      name: "",
      transport: "http_sse",
      endpoint: "",
      command: "",
      envText: "",
      enabled: true,
      message: ""
    });
  }

  function openEdit(item: ResourceConfig): void {
    if (!canWrite.value) return;
    Object.assign(form, {
      open: true,
      mode: "edit",
      configId: item.id,
      name: item.name,
      transport: item.mcp?.transport === "stdio" ? "stdio" : "http_sse",
      endpoint: item.mcp?.endpoint ?? "",
      command: item.mcp?.command ?? "",
      envText: item.mcp?.env ? JSON.stringify(item.mcp.env, null, 2) : "",
      enabled: item.enabled,
      message: ""
    });
  }

  function closeModal(): void {
    form.open = false;
  }

  async function saveConfig(): Promise<void> {
    const name = form.name.trim();
    if (name === "") {
      form.message = "名称不能为空";
      return;
    }

    const env = parseEnv(form.envText);
    if (env === "invalid") {
      form.message = "环境变量必须是 JSON 对象";
      return;
    }

    const payload = {
      transport: form.transport,
      endpoint: form.transport === "http_sse" ? form.endpoint.trim() : undefined,
      command: form.transport === "stdio" ? form.command.trim() : undefined,
      env: env || undefined
    };

    if (form.mode === "create") {
      await createWorkspaceResourceConfig({
        type: "mcp",
        name,
        enabled: form.enabled,
        mcp: payload
      });
    } else {
      await patchWorkspaceResourceConfig("mcp", form.configId, {
        name,
        enabled: form.enabled,
        mcp: payload
      });
    }

    form.open = false;
  }

  async function connect(item: ResourceConfig): Promise<void> {
    const result = await connectWorkspaceMcpConfig(item.id);
    if (!result) return;
    form.message = `${item.name}: ${result.status} tools=${result.tools.length} ${result.message}`;
  }

  function getConnectResult(configId: string) {
    return resourceStore.mcpConnectResultsByConfigId[configId] ?? null;
  }

  function connectStatusClass(status: string): string {
    return status === "connected" ? "enabled" : "disabled";
  }

  function connectSuggestion(errorCode?: string): string {
    if (!errorCode) return "";
    if (errorCode === "missing_command") {
      return "请检查 stdio command 是否可执行。";
    }
    if (errorCode === "invalid_endpoint") {
      return "请检查 SSE endpoint 地址格式。";
    }
    if (errorCode === "handshake_failed") {
      return "握手失败，请确认 MCP 服务是否运行。";
    }
    if (errorCode === "tools_list_failed") {
      return "tools/list 失败，请检查服务端协议实现。";
    }
    if (errorCode.startsWith("http_")) {
      return "服务返回异常状态，请检查鉴权与网关。";
    }
    return "请检查 MCP 配置并重试。";
  }

  async function toggleEnabled(item: ResourceConfig): Promise<void> {
    await patchWorkspaceResourceConfig("mcp", item.id, { enabled: !item.enabled });
  }

  async function removeConfig(item: ResourceConfig): Promise<void> {
    if (!window.confirm(`确认删除 MCP ${item.name} ?`)) {
      return;
    }
    await deleteWorkspaceResourceConfig("mcp", item.id);
  }

  async function openExportModal(): Promise<void> {
    await refreshWorkspaceMcpExport();
    form.jsonModalOpen = true;
  }

  function closeExportModal(): void {
    form.jsonModalOpen = false;
  }

  function formatTime(value?: string): string {
    if (!value) return "-";
    return new Date(value).toLocaleString();
  }

  return {
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
    refreshResourceConfigsByType,
    removeConfig,
    resourceStore,
    saveConfig,
    toggleEnabled
  };
}

function parseEnv(raw: string): Record<string, string> | null | "invalid" {
  const source = raw.trim();
  if (source === "") return null;
  try {
    const parsed = JSON.parse(source) as Record<string, unknown>;
    if (typeof parsed !== "object" || parsed === null) {
      return "invalid";
    }
    const output: Record<string, string> = {};
    for (const [key, value] of Object.entries(parsed)) {
      output[key] = String(value);
    }
    return output;
  } catch {
    return "invalid";
  }
}
