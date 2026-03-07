import { computed, reactive, watch } from "vue";

import { deleteProject, importProjectByDirectory, projectStore, refreshProjects, updateProjectBinding } from "@/modules/project/store";
import { refreshModelCatalog, refreshResourceConfigsByType, refreshWorkspaceProjectBindings, resourceStore } from "@/modules/resource/store";
import { getProjectConfig } from "@/modules/resource/views/projectConfigDefaults";
import { authStore } from "@/shared/stores/authStore";
import { formatTokenCompact, formatTokenUsageWithThreshold } from "@/shared/utils/tokenDisplay";
import { workspaceStore } from "@/shared/stores/workspaceStore";
import type { ProjectConfig } from "@/shared/types/api";

const columns = [
  { key: "name", label: "项目" },
  { key: "repoPath", label: "目录" },
  { key: "tokenUsage", label: "Token 用量" },
  { key: "modelCount", label: "模型绑定" },
  { key: "defaultModelId", label: "默认模型" },
  { key: "resourceSummary", label: "规则/技能/MCP" },
  { key: "actions", label: "动作" }
];

type OptionItem = { id: string; name: string };

export function useWorkspaceProjectConfigView() {
  const canWrite = computed(() => workspaceStore.mode === "local" || authStore.capabilities.resource_write);

  const form = reactive({
    open: false,
    projectId: "",
    projectName: "",
    modelIds: [] as string[],
    defaultModelId: "",
    tokenThresholdInput: "",
    modelTokenThresholdInputs: {} as Record<string, string>,
    ruleIds: [] as string[],
    skillIds: [] as string[],
    mcpIds: [] as string[],
    message: ""
  });

  const projectBindingByID = computed(
    () => new Map(resourceStore.projectBindings.map((item) => [item.project_id, item] as const))
  );

  const projectRows = computed(() => {
    return projectStore.projects.map((project) => {
      const binding = projectBindingByID.value.get(project.id);
      const config = binding?.config;
      const projectTokenThreshold =
        toPositiveInteger(project.token_threshold) ?? toPositiveInteger(config?.token_threshold) ?? null;
      const projectTokensTotal = project.tokens_total ?? binding?.tokens_total;

      return {
        id: project.id,
        name: project.name,
        repoPath: project.repo_path,
        tokenUsage: formatTokenUsageWithThreshold(projectTokensTotal, projectTokenThreshold),
        modelCount: normalizeModelBindingIDs(config?.model_config_ids ?? []).length,
        defaultModelId: resolveModelBindingDisplayName(config?.default_model_config_id ?? "") || "-",
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

  const modelOptions = computed<OptionItem[]>(() => {
    return resourceStore.models.items
      .map((item) => {
        const configID = item.id.trim();
        const modelID = item.model?.model_id?.trim() ?? "";
        if (configID === "" || modelID === "") {
          return null;
        }
        const vendor = item.model?.vendor?.trim() ?? "";
        const displayName = item.name?.trim() || (vendor ? `${vendor} / ${modelID}` : modelID);
        const suffix = item.enabled ? "" : " (Disabled)";
        return {
          id: configID,
          name: `${displayName}${suffix}`
        };
      })
      .filter((item): item is OptionItem => item !== null);
  });

  const ruleOptions = computed(() => resourceStore.rules.items.map((item) => ({ id: item.id, name: item.name })));
  const skillOptions = computed(() => resourceStore.skills.items.map((item) => ({ id: item.id, name: item.name })));
  const mcpOptions = computed(() => resourceStore.mcps.items.map((item) => ({ id: item.id, name: item.name })));

  const defaultModelOptions = computed(() =>
    form.modelIds
      .map((id) => modelOptions.value.find((item) => item.id === id))
      .filter((item): item is OptionItem => Boolean(item))
      .map((item) => ({ value: item.id, label: item.name }))
  );

  const activeProjectBinding = computed(() => {
    if (form.projectId.trim() === "") {
      return null;
    }
    return projectBindingByID.value.get(form.projectId) ?? null;
  });

  const projectUsageSummary = computed(() => {
    const usage = activeProjectBinding.value;
    const total = usage?.tokens_total;
    const threshold = parseThresholdInputLoose(form.tokenThresholdInput);
    return formatTokenUsageWithThreshold(total, threshold);
  });

  const projectUsageBreakdown = computed(() => {
    const usage = activeProjectBinding.value;
    return {
      input: formatTokenCompact(usage?.tokens_in_total),
      output: formatTokenCompact(usage?.tokens_out_total),
      total: formatTokenCompact(usage?.tokens_total)
    };
  });

  const projectModelUsageRows = computed(() => {
    const usageByModel = activeProjectBinding.value?.model_token_usage_by_config_id ?? {};
    return form.modelIds.map((modelConfigID) => {
      const modelOption = modelOptions.value.find((item) => item.id === modelConfigID);
      const usageItem = usageByModel[modelConfigID];
      const modelThreshold = parseThresholdInputLoose(form.modelTokenThresholdInputs[modelConfigID] ?? "");
      return {
        id: modelConfigID,
        name: modelOption?.name ?? modelConfigID,
        usageText: formatTokenUsageWithThreshold(usageItem?.tokens_total, modelThreshold)
      };
    });
  });

  watch(
    () => workspaceStore.currentWorkspaceId,
    async () => {
      await reload();
    },
    { immediate: true }
  );

  watch(
    () => [...form.modelIds],
    (modelIDs) => {
      const normalizedIDs = modelIDs.map((item) => item.trim()).filter((item) => item !== "");
      const allowedIDs = new Set(normalizedIDs);
      for (const modelConfigID of Object.keys(form.modelTokenThresholdInputs)) {
        if (!allowedIDs.has(modelConfigID)) {
          delete form.modelTokenThresholdInputs[modelConfigID];
        }
      }
      for (const modelConfigID of normalizedIDs) {
        if (!(modelConfigID in form.modelTokenThresholdInputs)) {
          form.modelTokenThresholdInputs[modelConfigID] = "";
        }
      }
      if (form.defaultModelId !== "" && !allowedIDs.has(form.defaultModelId)) {
        form.defaultModelId = "";
      }
    }
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

  async function removeProjectById(projectId: string): Promise<void> {
    if (!canWrite.value) {
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
    const normalizedModelIDs = normalizeModelBindingIDs(config.model_config_ids);
    const normalizedDefaultModelID = normalizeModelBindingID(config.default_model_config_id ?? "");
    const modelThresholdInputs = createModelTokenThresholdInputs(normalizedModelIDs, config.model_token_thresholds);

    Object.assign(form, {
      open: true,
      projectId,
      projectName: project.name,
      modelIds: normalizedModelIDs,
      defaultModelId: normalizedDefaultModelID,
      tokenThresholdInput: toPositiveInteger(config.token_threshold) ? String(config.token_threshold) : "",
      modelTokenThresholdInputs: modelThresholdInputs,
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
      assignFieldList(
        field,
        list.filter((item) => item !== id)
      );
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

    const tokenThreshold = parseThresholdInputStrict(form.tokenThresholdInput, "项目 Token 阀值");
    if (tokenThreshold.error) {
      form.message = tokenThreshold.error;
      return;
    }

    const modelTokenThresholds: Record<string, number> = {};
    for (const modelConfigID of normalizeModelBindingIDs(form.modelIds)) {
      const modelOption = modelOptions.value.find((item) => item.id === modelConfigID);
      const thresholdInput = form.modelTokenThresholdInputs[modelConfigID] ?? "";
      const parsedThreshold = parseThresholdInputStrict(
        thresholdInput,
        `${modelOption?.name ?? modelConfigID} Token 阀值`
      );
      if (parsedThreshold.error) {
        form.message = parsedThreshold.error;
        return;
      }
      if (parsedThreshold.value !== null) {
        modelTokenThresholds[modelConfigID] = parsedThreshold.value;
      }
    }

    const payload: Omit<ProjectConfig, "project_id" | "updated_at"> = {
      model_config_ids: [...form.modelIds],
      default_model_config_id: form.defaultModelId || null,
      token_threshold: tokenThreshold.value ?? undefined,
      model_token_thresholds: modelTokenThresholds,
      rule_ids: [...form.ruleIds],
      skill_ids: [...form.skillIds],
      mcp_ids: [...form.mcpIds]
    };

    form.message = "";
    const updated = await updateProjectBinding(form.projectId, payload);
    if (!updated) {
      form.message = projectStore.error.trim() || "保存失败，请检查项目配置";
      return;
    }
    await refreshWorkspaceProjectBindings();
    await refreshProjects();
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
    const normalizedSelector = id.trim();
    if (normalizedSelector === "") {
      return "";
    }

    const byConfigID = resourceStore.models.items.find((item) => item.id.trim() === normalizedSelector);
    if (byConfigID) {
      return byConfigID.id.trim();
    }

    return normalizedSelector;
  }

  function resolveModelBindingDisplayName(id: string): string {
    const normalizedConfigID = normalizeModelBindingID(id);
    if (normalizedConfigID === "") {
      return "";
    }
    const option = modelOptions.value.find((item) => item.id === normalizedConfigID);
    if (option) {
      return option.name;
    }
    return normalizedConfigID;
  }

  function createModelTokenThresholdInputs(
    modelConfigIDs: string[],
    sourceThresholds: ProjectConfig["model_token_thresholds"]
  ): Record<string, string> {
    const output: Record<string, string> = {};
    for (const modelConfigID of modelConfigIDs) {
      output[modelConfigID] = "";
    }
    const input = sourceThresholds ?? {};
    for (const [rawModelConfigID, rawThreshold] of Object.entries(input)) {
      const modelConfigID = normalizeModelBindingID(rawModelConfigID);
      const threshold = toPositiveInteger(rawThreshold);
      if (modelConfigID === "" || threshold === null || !(modelConfigID in output)) {
        continue;
      }
      output[modelConfigID] = String(threshold);
    }
    return output;
  }

  function parseThresholdInputStrict(
    input: string,
    fieldLabel: string
  ): { value: number | null; error: string | null } {
    const trimmed = input.trim();
    if (trimmed === "") {
      return { value: null, error: null };
    }
    if (!/^[0-9]+$/.test(trimmed)) {
      return {
        value: null,
        error: `${fieldLabel} 必须为正整数，留空表示不限`
      };
    }
    const parsed = Number.parseInt(trimmed, 10);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      return {
        value: null,
        error: `${fieldLabel} 必须为正整数，留空表示不限`
      };
    }
    return { value: parsed, error: null };
  }

  function parseThresholdInputLoose(input: string): number | null {
    const trimmed = input.trim();
    if (!/^[0-9]+$/.test(trimmed)) {
      return null;
    }
    return Number.parseInt(trimmed, 10);
  }

  function toPositiveInteger(value: unknown): number | null {
    if (typeof value !== "number" || !Number.isFinite(value) || value <= 0) {
      return null;
    }
    return Math.trunc(value);
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
    projectUsageBreakdown,
    projectUsageSummary,
    projectModelUsageRows,
    tableEmptyText,
    removeProjectById,
    resourceStore,
    ruleOptions,
    saveProjectBinding,
    skillOptions,
    toggleListItem
  };
}
