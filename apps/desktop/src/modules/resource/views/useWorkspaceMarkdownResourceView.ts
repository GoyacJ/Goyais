import { computed, reactive, watch } from "vue";

import {
  createWorkspaceResourceConfig,
  deleteWorkspaceResourceConfig,
  loadNextResourceConfigsPage,
  loadPreviousResourceConfigsPage,
  patchWorkspaceResourceConfig,
  refreshResourceConfigsByType,
  resourceStore,
  setResourceEnabledFilter,
  setResourceSearch
} from "@/modules/resource/store";
import { authStore } from "@/shared/stores/authStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import type { ResourceConfig } from "@/shared/types/api";

type MarkdownResourceType = "rule" | "skill";

const enabledOptions = [
  { value: "all", label: "全部" },
  { value: "enabled", label: "仅启用" },
  { value: "disabled", label: "仅停用" }
];

export function useWorkspaceMarkdownResourceView(type: MarkdownResourceType) {
  const canWrite = computed(() => workspaceStore.mode === "local" || authStore.capabilities.resource_write);
  const listState = computed(() => (type === "rule" ? resourceStore.rules : resourceStore.skills));

  const enabledFilterModel = computed({
    get: () => listState.value.enabledFilter,
    set: (value) => {
      setResourceEnabledFilter(type, value as "all" | "enabled" | "disabled");
      void refreshResourceConfigsByType(type);
    }
  });

  const form = reactive({
    open: false,
    mode: "create" as "create" | "edit",
    configId: "",
    name: "",
    content: "",
    enabled: true,
    message: ""
  });

  watch(
    () => workspaceStore.currentWorkspaceId,
    async () => {
      await refreshResourceConfigsByType(type);
    },
    { immediate: true }
  );

  function onSearch(value: string): void {
    setResourceSearch(type, value);
    void refreshResourceConfigsByType(type);
  }

  function openCreate(): void {
    if (!canWrite.value) return;
    Object.assign(form, {
      open: true,
      mode: "create",
      configId: "",
      name: "",
      content: "",
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
      content: type === "rule" ? item.rule?.content ?? "" : item.skill?.content ?? "",
      enabled: item.enabled,
      message: ""
    });
  }

  function closeModal(): void {
    form.open = false;
  }

  async function saveConfig(): Promise<void> {
    const name = form.name.trim();
    const content = form.content.trim();
    if (name === "" || content === "") {
      form.message = "名称和内容不能为空";
      return;
    }

    if (form.mode === "create") {
      if (type === "rule") {
        await createWorkspaceResourceConfig({ type, name, enabled: form.enabled, rule: { content } });
      } else {
        await createWorkspaceResourceConfig({ type, name, enabled: form.enabled, skill: { content } });
      }
    } else if (type === "rule") {
      await patchWorkspaceResourceConfig(type, form.configId, { name, enabled: form.enabled, rule: { content } });
    } else {
      await patchWorkspaceResourceConfig(type, form.configId, { name, enabled: form.enabled, skill: { content } });
    }

    form.open = false;
  }

  async function toggleEnabled(item: ResourceConfig): Promise<void> {
    await patchWorkspaceResourceConfig(type, item.id, { enabled: !item.enabled });
  }

  async function removeConfig(item: ResourceConfig): Promise<void> {
    if (!window.confirm(`确认删除 ${item.name} ?`)) {
      return;
    }
    await deleteWorkspaceResourceConfig(type, item.id);
  }

  function formatTime(value: string): string {
    return new Date(value).toLocaleString();
  }

  return {
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
  };
}
