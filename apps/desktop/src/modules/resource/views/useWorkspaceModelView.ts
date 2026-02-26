import { computed, reactive, watch } from "vue";

import {
  createWorkspaceResourceConfig,
  deleteWorkspaceResourceConfig,
  loadNextResourceConfigsPage,
  loadPreviousResourceConfigsPage,
  patchWorkspaceResourceConfig,
  refreshModelCatalog,
  reloadWorkspaceModelCatalog,
  refreshResourceConfigsByType,
  resourceStore,
  setResourceEnabledFilter,
  setResourceSearch,
  testWorkspaceModelConfig
} from "@/modules/resource/store";
import { modelViewColumns, modelEnabledOptions } from "@/modules/resource/views/workspaceModelView.constants";
import { resolveDefaultEndpointKey } from "@/modules/resource/views/workspaceModelView.vendor";
import { authStore } from "@/shared/stores/authStore";
import { showToast, type ToastTone } from "@/shared/stores/toastStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import type { ModelVendorName, ResourceConfig } from "@/shared/types/api";

export function useWorkspaceModelView() {
  const canWrite = computed(() => workspaceStore.mode === "local" || authStore.capabilities.resource_write);
  const catalogVendors = computed(() => resourceStore.catalog?.vendors ?? []);
  const vendorOptions = computed(() => catalogVendors.value.map((item) => ({ value: item.name, label: item.name })));

  const form = reactive({
    open: false,
    mode: "create" as "create" | "edit",
    configId: "",
    vendor: "" as ModelVendorName | "",
    selectedCatalogModel: "",
    baseUrl: "",
    baseUrlKey: "",
    apiKey: "",
    apiKeyHint: "",
    timeoutMs: "",
    enabled: true,
    testMessage: ""
  });

  const selectedVendor = computed(() => catalogVendors.value.find((item) => item.name === form.vendor) ?? null);
  const vendorModelOptions = computed(() => {
    const models = selectedVendor.value?.models ?? [];
    const selectedID = form.selectedCatalogModel;
    return models
      .filter((item) => item.enabled || item.id === selectedID)
      .map((item) => ({
        value: item.id,
        label: item.enabled ? item.label : `${item.label} (Disabled)`
      }));
  });
  const vendorEndpointOptions = computed(() => {
    const entries = Object.entries(selectedVendor.value?.base_urls ?? {});
    return entries.map(([key, value]) => ({ value: key, label: `${key} (${value})` }));
  });
  const showVendorEndpointSelector = computed(() => form.vendor !== "Local" && vendorEndpointOptions.value.length > 0);
  const showLocalBaseURL = computed(() => form.vendor === "Local");
  const selectedVendorBaseURL = computed(() => selectedVendor.value?.base_url ?? "");

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

  const deleteConfirm = reactive({
    open: false,
    configId: "",
    modelText: ""
  });

  watch(
    () => workspaceStore.currentWorkspaceId,
    async () => {
      const loadCatalogTask = canWrite.value ? reloadWorkspaceModelCatalog("page_open") : refreshModelCatalog();
      await Promise.all([loadCatalogTask, refreshResourceConfigsByType("model")]);
    },
    { immediate: true }
  );

  watch(
    () => form.vendor,
    (vendor) => {
      const vendorConfig = catalogVendors.value.find((item) => item.name === vendor);
      const selectableModels = (vendorConfig?.models ?? []).filter((item) => item.enabled || item.id === form.selectedCatalogModel);
      const firstModel = selectableModels[0]?.id ?? "";
      const modelExists = selectableModels.some((item) => item.id === form.selectedCatalogModel);
      if (!modelExists) {
        form.selectedCatalogModel = firstModel;
      }

      if (vendor === "Local") {
        if (form.baseUrl.trim() === "") {
          form.baseUrl = vendorConfig?.base_url ?? "";
        }
        form.baseUrlKey = "";
        return;
      }

      form.baseUrl = "";
      const baseURLs = vendorConfig?.base_urls ?? {};
      if (!(form.baseUrlKey in baseURLs)) {
        form.baseUrlKey = resolveDefaultEndpointKey(vendorConfig);
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
    const firstModel = (firstVendor?.models ?? []).find((item) => item.enabled)?.id ?? "";
    Object.assign(form, {
      open: true,
      mode: "create",
      configId: "",
      vendor: (firstVendor?.name ?? "") as ModelVendorName | "",
      selectedCatalogModel: firstModel,
      baseUrl: firstVendor?.name === "Local" ? firstVendor.base_url : "",
      baseUrlKey: firstVendor?.name === "Local" ? "" : resolveDefaultEndpointKey(firstVendor),
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
      baseUrlKey: item.model?.base_url_key ?? "",
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
      base_url_key: vendor === "Local" ? undefined : form.baseUrlKey.trim() || undefined,
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

  function showTestNotice(tone: ToastTone, message: string): void {
    showToast({
      key: "workspace-model-test",
      tone,
      message
    });
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
    columns: modelViewColumns,
    enabledFilterModel,
    enabledOptions: modelEnabledOptions,
    form,
    tableEmptyText,
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
    vendorEndpointOptions,
    showVendorEndpointSelector,
    showLocalBaseURL,
    selectedVendorBaseURL
  };
}
