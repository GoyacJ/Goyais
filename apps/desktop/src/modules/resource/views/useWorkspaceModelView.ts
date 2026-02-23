import { computed, reactive, watch } from "vue";

import {
  createWorkspaceResourceConfig,
  deleteWorkspaceResourceConfig,
  loadNextResourceConfigsPage,
  loadPreviousResourceConfigsPage,
  patchWorkspaceResourceConfig,
  refreshModelCatalog,
  refreshResourceConfigsByType,
  resourceStore,
  setResourceEnabledFilter,
  setResourceSearch,
  testWorkspaceModelConfig
} from "@/modules/resource/store";
import { authStore } from "@/shared/stores/authStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import type { ModelVendorName, ResourceConfig } from "@/shared/types/api";

const columns = [
  { key: "vendor", label: "厂商" },
  { key: "model", label: "模型" },
  { key: "enabled", label: "状态" },
  { key: "updated", label: "更新时间" },
  { key: "actions", label: "动作" }
];

const enabledOptions = [
  { value: "all", label: "全部" },
  { value: "enabled", label: "仅启用" },
  { value: "disabled", label: "仅停用" }
];

export function useWorkspaceModelView() {
  const canWrite = computed(() => workspaceStore.mode === "local" || authStore.capabilities.resource_write);
  const catalogVendors = computed(() => resourceStore.catalog?.vendors ?? []);
  const vendorOptions = computed(() => catalogVendors.value.map((item) => ({ value: item.name, label: item.name })));
  const vendorModelOptions = computed(() => {
    const vendor = catalogVendors.value.find((item) => item.name === form.vendor);
    return (vendor?.models ?? []).map((item) => ({ value: item.id, label: item.label }));
  });
  const showLocalBaseURL = computed(() => form.vendor === "Local");
  const selectedVendorBaseURL = computed(() => {
    const vendor = catalogVendors.value.find((item) => item.name === form.vendor);
    return vendor?.base_url ?? "";
  });

  const enabledFilterModel = computed({
    get: () => resourceStore.models.enabledFilter,
    set: (value) => {
      setResourceEnabledFilter("model", value as "all" | "enabled" | "disabled");
      void refreshResourceConfigsByType("model");
    }
  });

  const tableEmptyText = computed(() => {
    if (resourceStore.error.trim() !== "") {
      return resourceStore.error;
    }
    return "暂无数据";
  });

  const form = reactive({
    open: false,
    mode: "create" as "create" | "edit",
    configId: "",
    vendor: "" as ModelVendorName | "",
    selectedCatalogModel: "",
    baseUrl: "",
    apiKey: "",
    apiKeyHint: "",
    timeoutMs: "",
    enabled: true,
    testMessage: ""
  });
  const testNotice = reactive({
    open: false,
    tone: "info" as "error" | "warning" | "info" | "403" | "disconnected" | "retrying",
    message: ""
  });
  const deleteConfirm = reactive({
    open: false,
    configId: "",
    modelText: ""
  });
  let testNoticeTimer: ReturnType<typeof setTimeout> | null = null;

  watch(
    () => workspaceStore.currentWorkspaceId,
    async () => {
      await Promise.all([refreshModelCatalog(), refreshResourceConfigsByType("model")]);
    },
    { immediate: true }
  );

  watch(
    () => form.vendor,
    (vendor) => {
      const vendorConfig = catalogVendors.value.find((item) => item.name === vendor);
      const firstModel = vendorConfig?.models[0]?.id ?? "";
      const modelExists = vendorConfig?.models.some((item) => item.id === form.selectedCatalogModel) ?? false;
      if (!modelExists) {
        form.selectedCatalogModel = firstModel;
      }
      if (vendor === "Local" && form.baseUrl.trim() === "") {
        form.baseUrl = vendorConfig?.base_url ?? "";
      }
      if (vendor !== "Local") {
        form.baseUrl = "";
      }
    }
  );

  function onSearch(value: string): void {
    setResourceSearch("model", value);
    void refreshResourceConfigsByType("model");
  }

  function openCreate(): void {
    if (!canWrite.value) return;
    const firstVendor = catalogVendors.value[0];
    Object.assign(form, {
      open: true,
      mode: "create",
      configId: "",
      vendor: (firstVendor?.name ?? "") as ModelVendorName | "",
      selectedCatalogModel: firstVendor?.models[0]?.id ?? "",
      baseUrl: firstVendor?.name === "Local" ? firstVendor.base_url : "",
      apiKey: "",
      apiKeyHint: "",
      timeoutMs: "",
      enabled: true
    });
  }

  function openEdit(item: ResourceConfig): void {
    if (!canWrite.value) return;
    const vendor = (item.model?.vendor ?? "") as ModelVendorName | "";
    Object.assign(form, {
      open: true,
      mode: "edit",
      configId: item.id,
      vendor,
      selectedCatalogModel: item.model?.model_id ?? "",
      baseUrl: item.model?.base_url ?? "",
      apiKey: "",
      apiKeyHint: item.model?.api_key_masked ? `当前: ${item.model.api_key_masked}（不填写将保留旧值）` : "",
      timeoutMs: item.model?.timeout_ms ? String(item.model.timeout_ms) : "",
      enabled: item.enabled
    });
  }

  function closeModal(): void {
    form.open = false;
  }

  async function saveConfig(): Promise<void> {
    const vendor = form.vendor;
    const modelID = form.selectedCatalogModel.trim();
    if (vendor === "" || modelID === "") {
      form.testMessage = "请选择厂商与模型";
      return;
    }

    const timeout = Number.parseInt(form.timeoutMs, 10);
    const model = {
      vendor,
      model_id: modelID,
      base_url: vendor === "Local" ? form.baseUrl.trim() || undefined : undefined,
      api_key: form.apiKey.trim() || undefined,
      timeout_ms: Number.isNaN(timeout) ? undefined : timeout
    };

    if (form.mode === "create") {
      await createWorkspaceResourceConfig({ type: "model", enabled: form.enabled, model });
    } else {
      await patchWorkspaceResourceConfig("model", form.configId, { enabled: form.enabled, model });
    }
    form.open = false;
  }

  async function toggleEnabled(item: ResourceConfig): Promise<void> {
    await patchWorkspaceResourceConfig("model", item.id, { enabled: !item.enabled });
  }

  async function runModelTest(item: ResourceConfig): Promise<void> {
    const result = await testWorkspaceModelConfig(item.id);
    if (!result) {
      showTestNotice("error", resourceStore.error || "模型测试失败");
      return;
    }
    const modelText = `${item.model?.vendor ?? "-"} / ${item.model?.model_id ?? "-"}`;
    const statusText = result.status === "success" ? "成功" : "失败";
    const tone = result.status === "success" ? "info" : "error";
    const detail = result.error_code ? ` / ${result.error_code}` : "";
    showTestNotice(tone, `${modelText} 测试${statusText}，${result.latency_ms}ms${detail}：${result.message}`);
  }

  function showTestNotice(tone: "error" | "warning" | "info" | "403" | "disconnected" | "retrying", message: string): void {
    testNotice.tone = tone;
    testNotice.message = message;
    testNotice.open = true;
    if (testNoticeTimer) {
      clearTimeout(testNoticeTimer);
    }
    testNoticeTimer = setTimeout(() => {
      testNotice.open = false;
    }, 3200);
  }

  function removeConfig(item: ResourceConfig): void {
    deleteConfirm.open = true;
    deleteConfirm.configId = item.id;
    deleteConfirm.modelText = `${item.model?.vendor ?? "-"} / ${item.model?.model_id ?? "-"}`;
  }

  function closeDeleteConfirm(): void {
    deleteConfirm.open = false;
    deleteConfirm.configId = "";
    deleteConfirm.modelText = "";
  }

  async function confirmRemoveConfig(): Promise<void> {
    if (!deleteConfirm.configId) {
      return;
    }
    const modelText = deleteConfirm.modelText;
    const configID = deleteConfirm.configId;
    const removed = await deleteWorkspaceResourceConfig("model", configID);
    if (!removed) {
      showTestNotice("error", resourceStore.error || `删除失败：${modelText}`);
      return;
    }
    closeDeleteConfirm();
    showTestNotice("info", `已删除模型配置：${modelText}`);
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
    tableEmptyText,
    testNotice,
    deleteConfirm,
    onSearch,
    openCreate,
    openEdit,
    closeModal,
    saveConfig,
    toggleEnabled,
    runModelTest,
    removeConfig,
    closeDeleteConfirm,
    confirmRemoveConfig,
    formatTime,
    loadNextResourceConfigsPage,
    loadPreviousResourceConfigsPage,
    resourceStore,
    vendorModelOptions,
    vendorOptions,
    showLocalBaseURL,
    selectedVendorBaseURL
  };
}
