import { computed, type ComputedRef } from "vue";

import { projectStore } from "@/modules/project/store";
import { resourceStore } from "@/modules/resource/store";
import type { ConversationRuntime } from "@/modules/conversation/store/state";
import type { Conversation, Project } from "@/shared/types/api";

type MainScreenModelingInput = {
  activeProject: ComputedRef<Project | undefined>;
  activeConversation: ComputedRef<Conversation | undefined>;
  runtime: ComputedRef<ConversationRuntime | undefined>;
};

export function useMainScreenModeling(input: MainScreenModelingInput) {
  const modelLabelByConfigID = computed(() => {
    const map = new Map<string, string>();
    for (const item of resourceStore.models.items) {
      const configID = item.id.trim();
      const modelID = item.model?.model_id?.trim() ?? "";
      if (configID === "" || modelID === "") {
        continue;
      }
      const vendor = item.model?.vendor?.trim() ?? "-";
      map.set(configID, `${vendor} / ${modelID}`);
    }
    return map;
  });

  const modelLabelByModelID = computed(() => {
    const map = new Map<string, string>();
    for (const item of resourceStore.models.items) {
      if (!item.enabled) {
        continue;
      }
      const modelID = item.model?.model_id?.trim() ?? "";
      if (modelID === "" || map.has(modelID)) {
        continue;
      }
      const vendor = item.model?.vendor?.trim() ?? "-";
      map.set(modelID, `${vendor} / ${modelID}`);
    }
    return map;
  });

  const enabledModelIDs = computed(() => {
    const items: string[] = [];
    for (const item of resourceStore.models.items) {
      if (!item.enabled) {
        continue;
      }
      const configID = item.id.trim();
      const modelID = item.model?.model_id?.trim() ?? "";
      if (configID === "" || modelID === "" || items.includes(configID)) {
        continue;
      }
      items.push(configID);
    }
    return items;
  });
  const availableModelIDs = computed(() => new Set(enabledModelIDs.value));

  function resolveSemanticModelID(raw: string): string {
    const normalized = raw.trim();
    if (normalized === "") {
      return "";
    }
    if (modelLabelByConfigID.value.has(normalized)) {
      return normalized;
    }

    const byEnabledModelID = resourceStore.models.items.find(
      (item) => item.enabled && (item.model?.model_id?.trim() ?? "") === normalized
    );
    if (byEnabledModelID) {
      return byEnabledModelID.id.trim();
    }

    const byModelID = resourceStore.models.items.find((item) => (item.model?.model_id?.trim() ?? "") === normalized);
    if (byModelID) {
      return byModelID.id.trim();
    }

    return normalized;
  }

  function resolveModelLabel(configOrModelID: string): string {
    const normalized = configOrModelID.trim();
    if (normalized === "") {
      return "";
    }
    const byConfigID = modelLabelByConfigID.value.get(normalized);
    if (byConfigID) {
      return byConfigID;
    }
    const byModelID = modelLabelByModelID.value.get(normalized);
    if (byModelID) {
      return byModelID;
    }
    return normalized;
  }

  const modelOptions = computed<Array<{ value: string; label: string }>>(() => {
    const project = input.activeProject.value;
    const projectConfig = project ? projectStore.projectConfigsByProjectId[project.id] : undefined;

    const configuredModelIDs = [
      ...(projectConfig?.model_ids ?? []),
      projectConfig?.default_model_id ?? ""
    ]
      .map((value) => resolveSemanticModelID(value))
      .filter((value, index, source) => value !== "" && source.indexOf(value) === index)
      .filter((value) => availableModelIDs.value.has(value));

    if (configuredModelIDs.length > 0) {
      return configuredModelIDs.map((value) => ({ value, label: resolveModelLabel(value) }));
    }

    return enabledModelIDs.value.map((value) => ({ value, label: resolveModelLabel(value) }));
  });

  const activeModelId = computed(() => {
    const runtimeModelID = resolveSemanticModelID(input.runtime.value?.modelId ?? input.activeConversation.value?.model_id ?? "");
    if (runtimeModelID !== "" && modelOptions.value.some((item) => item.value === runtimeModelID)) {
      return runtimeModelID;
    }
    return modelOptions.value[0]?.value ?? "";
  });

  return {
    modelOptions,
    activeModelId,
    enabledModelIDs,
    availableModelIDs,
    resolveSemanticModelID,
    resolveModelLabel
  };
}
