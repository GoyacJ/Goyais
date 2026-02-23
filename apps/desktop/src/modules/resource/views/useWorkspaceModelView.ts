import { computed, reactive, watch } from "vue";

import {
  createWorkspaceResourceConfig,
  deleteWorkspaceResourceConfig,
  loadNextResourceConfigsPage,
  loadPreviousResourceConfigsPage,
  patchWorkspaceResourceConfig,
  refreshModelCatalog,
  refreshResourceConfigsByType,
  reloadWorkspaceModelCatalog,
  resourceStore,
  setResourceEnabledFilter,
  setResourceSearch,
  testWorkspaceModelConfig
} from "@/modules/resource/store";
import { authStore } from "@/shared/stores/authStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import type { ModelVendorName, ResourceConfig } from "@/shared/types/api";

const columns = [
  { key: "name", label: "名称" },
  { key: "vendor", label: "厂商" },
  { key: "model", label: "模型" },
  { key: "probe", label: "测试诊断" },
  { key: "enabled", label: "状态" },
  { key: "updated", label: "更新时间" },
  { key: "actions", label: "动作" }
];

const enabledOptions = [
  { value: "all", label: "全部" },
  { value: "enabled", label: "仅启用" },
  { value: "disabled", label: "仅停用" }
];

const fallbackVendors: Array<{ value: string; label: string }> = [
  { value: "OpenAI", label: "OpenAI" },
  { value: "Google", label: "Google" },
  { value: "Qwen", label: "Qwen" },
  { value: "Doubao", label: "Doubao" },
  { value: "Zhipu", label: "Zhipu" },
  { value: "MiniMax", label: "MiniMax" },
  { value: "Local", label: "Local" }
];

export function useWorkspaceModelView() {
  const canWrite = computed(() => workspaceStore.mode === "local" || authStore.capabilities.resource_write);
  const catalogVendors = computed(() => resourceStore.catalog?.vendors ?? []);
  const vendorOptions = computed(() => {
    const options = catalogVendors.value.map((item) => ({ value: item.name, label: item.name }));
    return options.length > 0 ? options : fallbackVendors;
  });
  const vendorModelOptions = computed(() => {
    const vendor = catalogVendors.value.find((item) => item.name === form.vendor);
    const options = (vendor?.models ?? []).map((item) => ({ value: item.id, label: `${item.label} (${item.id})` }));
    return [{ value: "", label: "请选择或手动输入" }, ...options];
  });

  const enabledFilterModel = computed({
    get: () => resourceStore.models.enabledFilter,
    set: (value) => {
      setResourceEnabledFilter("model", value as "all" | "enabled" | "disabled");
      void refreshResourceConfigsByType("model");
    }
  });

  const form = reactive({
    open: false,
    mode: "create" as "create" | "edit",
    configId: "",
    name: "",
    vendor: "OpenAI" as ModelVendorName,
    selectedCatalogModel: "",
    modelId: "",
    baseUrl: "",
    apiKey: "",
    apiKeyHint: "",
    timeoutMs: "",
    paramsText: "",
    enabled: true,
    testMessage: ""
  });

  watch(
    () => workspaceStore.currentWorkspaceId,
    async () => {
      await Promise.all([refreshModelCatalog(), refreshResourceConfigsByType("model")]);
    },
    { immediate: true }
  );

  watch(
    () => form.selectedCatalogModel,
    (value) => {
      if (value.trim() !== "") {
        form.modelId = value;
      }
    }
  );

  function onSearch(value: string): void {
    setResourceSearch("model", value);
    void refreshResourceConfigsByType("model");
  }

  function openCreate(): void {
    if (!canWrite.value) return;
    const firstVendor = (catalogVendors.value[0]?.name ?? "OpenAI") as ModelVendorName;
    Object.assign(form, {
      open: true,
      mode: "create",
      configId: "",
      name: "",
      vendor: firstVendor,
      selectedCatalogModel: "",
      modelId: "",
      baseUrl: "",
      apiKey: "",
      apiKeyHint: "",
      timeoutMs: "",
      paramsText: "",
      enabled: true
    });
  }

  function openEdit(item: ResourceConfig): void {
    if (!canWrite.value) return;
    Object.assign(form, {
      open: true,
      mode: "edit",
      configId: item.id,
      name: item.name,
      vendor: item.model?.vendor ?? "OpenAI",
      selectedCatalogModel: item.model?.model_id ?? "",
      modelId: item.model?.model_id ?? "",
      baseUrl: item.model?.base_url ?? "",
      apiKey: "",
      apiKeyHint: item.model?.api_key_masked ? `当前: ${item.model.api_key_masked}（不填写将保留旧值）` : "",
      timeoutMs: item.model?.timeout_ms ? String(item.model.timeout_ms) : "",
      paramsText: item.model?.params ? JSON.stringify(item.model.params, null, 2) : "",
      enabled: item.enabled
    });
  }

  function closeModal(): void {
    form.open = false;
  }

  async function saveConfig(): Promise<void> {
    const name = form.name.trim();
    const modelID = form.modelId.trim();
    if (name === "" || modelID === "") {
      form.testMessage = "名称和模型 ID 不能为空";
      return;
    }

    const parsedParams = parseParams(form.paramsText);
    if (parsedParams === "invalid") {
      form.testMessage = "Params 必须是合法 JSON";
      return;
    }

    const timeout = Number.parseInt(form.timeoutMs, 10);
    const model = {
      vendor: form.vendor,
      model_id: modelID,
      base_url: form.baseUrl.trim() || undefined,
      api_key: form.apiKey.trim() || undefined,
      timeout_ms: Number.isNaN(timeout) ? undefined : timeout,
      params: parsedParams || undefined
    };

    if (form.mode === "create") {
      await createWorkspaceResourceConfig({ type: "model", name, enabled: form.enabled, model });
    } else {
      await patchWorkspaceResourceConfig("model", form.configId, { name, enabled: form.enabled, model });
    }
    form.open = false;
  }

  async function toggleEnabled(item: ResourceConfig): Promise<void> {
    await patchWorkspaceResourceConfig("model", item.id, { enabled: !item.enabled });
  }

  async function runModelTest(item: ResourceConfig): Promise<void> {
    const result = await testWorkspaceModelConfig(item.id);
    if (!result) return;
    form.testMessage = `${item.name}: ${result.status} (${result.latency_ms}ms) ${result.message}`;
  }

  function getProbeResult(configId: string) {
    return resourceStore.modelTestResultsByConfigId[configId] ?? null;
  }

  function probeSuggestion(errorCode?: string): string {
    if (!errorCode) return "";
    if (errorCode === "missing_api_key") {
      return "请补充 API Key。";
    }
    if (errorCode === "invalid_base_url") {
      return "请检查 Base URL 与厂商网关。";
    }
    if (errorCode.startsWith("http_")) {
      return "请检查网络、鉴权与模型 ID 是否有效。";
    }
    if (errorCode === "request_failed") {
      return "请求失败，请检查网络连通性。";
    }
    return "请检查模型配置后重试。";
  }

  function probeStatusClass(status: string): string {
    return status === "success" ? "enabled" : "disabled";
  }

  async function removeConfig(item: ResourceConfig): Promise<void> {
    if (!window.confirm(`确认删除模型配置 ${item.name} ?`)) {
      return;
    }
    await deleteWorkspaceResourceConfig("model", item.id);
  }

  async function reloadCatalog(): Promise<void> {
    await reloadWorkspaceModelCatalog();
  }

  function formatTime(value: string): string {
    return new Date(value).toLocaleString();
  }

  return {
    canWrite,
    columns,
    enabledFilterModel,
    enabledOptions,
    form,
    onSearch,
    openCreate,
    openEdit,
    closeModal,
    saveConfig,
    toggleEnabled,
    runModelTest,
    removeConfig,
    reloadCatalog,
    getProbeResult,
    probeSuggestion,
    probeStatusClass,
    formatTime,
    loadNextResourceConfigsPage,
    loadPreviousResourceConfigsPage,
    resourceStore,
    vendorModelOptions,
    vendorOptions
  };
}

function parseParams(raw: string): Record<string, unknown> | null | "invalid" {
  const source = raw.trim();
  if (source === "") return null;
  try {
    const parsed = JSON.parse(source) as Record<string, unknown>;
    return typeof parsed === "object" && parsed !== null ? parsed : null;
  } catch {
    return "invalid";
  }
}
