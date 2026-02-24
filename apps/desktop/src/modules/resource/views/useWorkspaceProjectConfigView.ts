import { computed, reactive, watch } from "vue";

import { deleteProject, importProjectByDirectory, projectStore, refreshProjects, updateProjectBinding } from "@/modules/project/store";
import { refreshModelCatalog, refreshResourceConfigsByType, refreshWorkspaceProjectBindings, resourceStore } from "@/modules/resource/store";
import { authStore } from "@/shared/stores/authStore";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import type { ProjectConfig } from "@/shared/types/api";

const columns = [
  { key: "name", label: "项目" },
  { key: "repoPath", label: "目录" },
  { key: "modelCount", label: "模型绑定" },
  { key: "defaultModelId", label: "默认模型" },
  { key: "resourceSummary", label: "规则/技能/MCP" },
  { key: "actions", label: "动作" }
];

export function useWorkspaceProjectConfigView() {
  const canWrite = computed(() => workspaceStore.mode === "local" || authStore.capabilities.resource_write);

  const form = reactive({
    open: false,
    projectId: "",
    projectName: "",
    modelIds: [] as string[],
    defaultModelId: "",
    ruleIds: [] as string[],
    skillIds: [] as string[],
    mcpIds: [] as string[],
    message: ""
  });

  const projectRows = computed(() => {
    const byId = new Map(resourceStore.projectBindings.map((item) => [item.project_id, item] as const));
    return projectStore.projects.map((project) => {
      const binding = byId.get(project.id);
      const config = binding?.config;
      return {
        id: project.id,
        name: project.name,
        repoPath: project.repo_path,
        modelCount: normalizeModelBindingIDs(config?.model_ids ?? []).length,
        defaultModelId: resolveModelBindingDisplayName(config?.default_model_id ?? "") || "-",
        ruleCount: config?.rule_ids.length ?? 0,
        skillCount: config?.skill_ids.length ?? 0,
        mcpCount: config?.mcp_ids.length ?? 0,
        resourceSummary: `${config?.rule_ids.length ?? 0}/${config?.skill_ids.length ?? 0}/${config?.mcp_ids.length ?? 0}`
      };
    });
  });
  const tableEmptyText = computed(() => {
    if (projectStore.error.trim() !== "") {
      return projectStore.error;
    }
    if (resourceStore.error.trim() !== "") {
      return resourceStore.error;
    }
    return "当前工作区暂无项目";
  });

  const modelOptions = computed(() => {
    const labelsByModelID = new Map<string, string>();
    for (const item of resourceStore.models.items) {
      const modelID = item.model?.model_id?.trim() ?? "";
      if (modelID === "") {
        continue;
      }
      if (labelsByModelID.has(modelID)) {
        continue;
      }
      const vendor = item.model?.vendor?.trim() ?? "-";
      labelsByModelID.set(modelID, `${vendor} / ${modelID}`);
    }
    return Array.from(labelsByModelID.entries()).map(([id, name]) => ({ id, name }));
  });
  const ruleOptions = computed(() => resourceStore.rules.items.map((item) => ({ id: item.id, name: item.name })));
  const skillOptions = computed(() => resourceStore.skills.items.map((item) => ({ id: item.id, name: item.name })));
  const mcpOptions = computed(() => resourceStore.mcps.items.map((item) => ({ id: item.id, name: item.name })));

  const defaultModelOptions = computed(() =>
    form.modelIds
      .map((id) => modelOptions.value.find((item) => item.id === id))
      .filter((item): item is { id: string; name: string } => Boolean(item))
      .map((item) => ({ value: item.id, label: item.name }))
  );

  watch(
    () => workspaceStore.currentWorkspaceId,
    async () => {
      await reload();
    },
    { immediate: true }
  );

  async function reload(): Promise<void> {
    await Promise.all([
      refreshProjects(),
      refreshWorkspaceProjectBindings(),
      refreshResourceConfigsByType("model"),
      refreshResourceConfigsByType("rule"),
      refreshResourceConfigsByType("skill"),
      refreshResourceConfigsByType("mcp"),
      refreshModelCatalog()
    ]);
  }

  async function importDirectoryProject(repoPath: string): Promise<void> {
    if (!canWrite.value) {
      return;
    }

    const path = repoPath.trim();
    if (path === "") {
      form.message = "目录路径不能为空";
      return;
    }

    form.message = "";
    await importProjectByDirectory(path);
    await refreshWorkspaceProjectBindings();
  }

  async function removeProjectById(projectId: string, projectName: string): Promise<void> {
    if (!canWrite.value) {
      return;
    }

    if (!window.confirm(`确认移除项目 ${projectName} ?`)) {
      return;
    }
    await deleteProject(projectId);
    await refreshWorkspaceProjectBindings();
  }

  function openProjectBinding(projectId: string): void {
    const project = projectStore.projects.find((item) => item.id === projectId);
    if (!project) {
      return;
    }

    const config = getProjectConfig(projectId);
    const normalizedModelIDs = normalizeModelBindingIDs(config.model_ids);
    const normalizedDefaultModelID = normalizeModelBindingID(config.default_model_id ?? "");
    Object.assign(form, {
      open: true,
      projectId,
      projectName: project.name,
      modelIds: normalizedModelIDs,
      defaultModelId: normalizedDefaultModelID,
      ruleIds: [...config.rule_ids],
      skillIds: [...config.skill_ids],
      mcpIds: [...config.mcp_ids],
      message: ""
    });
  }

  function closeProjectBinding(): void {
    form.open = false;
  }

  function toggleListItem(field: "modelIds" | "ruleIds" | "skillIds" | "mcpIds", id: string): void {
    const list = resolveFieldList(field);
    const exists = list.includes(id);
    if (exists) {
      assignFieldList(field, list.filter((item) => item !== id));
      if (field === "modelIds" && form.defaultModelId === id) {
        form.defaultModelId = "";
      }
      return;
    }
    assignFieldList(field, [...list, id]);
    if (field === "modelIds" && form.defaultModelId === "") {
      form.defaultModelId = id;
    }
  }

  function resolveFieldList(field: "modelIds" | "ruleIds" | "skillIds" | "mcpIds"): string[] {
    if (field === "modelIds") {
      return form.modelIds;
    }
    if (field === "ruleIds") {
      return form.ruleIds;
    }
    if (field === "skillIds") {
      return form.skillIds;
    }
    return form.mcpIds;
  }

  function assignFieldList(field: "modelIds" | "ruleIds" | "skillIds" | "mcpIds", value: string[]): void {
    if (field === "modelIds") {
      form.modelIds = value;
      return;
    }
    if (field === "ruleIds") {
      form.ruleIds = value;
      return;
    }
    if (field === "skillIds") {
      form.skillIds = value;
      return;
    }
    form.mcpIds = value;
  }

  async function saveProjectBinding(): Promise<void> {
    if (form.projectId === "") {
      return;
    }

    if (form.defaultModelId !== "" && !form.modelIds.includes(form.defaultModelId)) {
      form.message = "默认模型必须属于已绑定模型";
      return;
    }

    const payload: Omit<ProjectConfig, "project_id" | "updated_at"> = {
      model_ids: [...form.modelIds],
      default_model_id: form.defaultModelId || null,
      rule_ids: [...form.ruleIds],
      skill_ids: [...form.skillIds],
      mcp_ids: [...form.mcpIds]
    };

    await updateProjectBinding(form.projectId, payload);
    await refreshWorkspaceProjectBindings();
    form.open = false;
  }

  function isChecked(field: "modelIds" | "ruleIds" | "skillIds" | "mcpIds", id: string): boolean {
    return form[field].includes(id);
  }

  function normalizeModelBindingIDs(modelIDs: string[]): string[] {
    const normalized = modelIDs
      .map((id) => normalizeModelBindingID(id))
      .filter((id, index, source) => id !== "" && source.indexOf(id) === index);
    return normalized;
  }

  function normalizeModelBindingID(id: string): string {
    const normalizedID = id.trim();
    if (normalizedID === "") {
      return "";
    }
    const byConfigID = resourceStore.models.items.find((item) => item.id === normalizedID);
    if (byConfigID?.model?.model_id) {
      return byConfigID.model.model_id.trim();
    }
    return normalizedID;
  }

  function resolveModelBindingDisplayName(id: string): string {
    const normalizedModelID = normalizeModelBindingID(id);
    if (normalizedModelID === "") {
      return "";
    }
    const option = modelOptions.value.find((item) => item.id === normalizedModelID);
    return option?.name ?? normalizedModelID;
  }

  return {
    canWrite,
    closeProjectBinding,
    columns,
    defaultModelOptions,
    form,
    importDirectoryProject,
    isChecked,
    mcpOptions,
    modelOptions,
    openProjectBinding,
    projectRows,
    tableEmptyText,
    removeProjectById,
    resourceStore,
    ruleOptions,
    saveProjectBinding,
    skillOptions,
    toggleListItem
  };
}

function getProjectConfig(projectId: string): ProjectConfig {
  const config = projectStore.projectConfigsByProjectId[projectId];
  if (config) {
    return config;
  }

  return {
    project_id: projectId,
    model_ids: [],
    default_model_id: null,
    rule_ids: [],
    skill_ids: [],
    mcp_ids: [],
    updated_at: new Date().toISOString()
  };
}
