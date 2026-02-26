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
  const enabledModelByConfigID = computed(() => {
    const map = new Map<string, { label: string }>();
    for (const item of resourceStore.models.items) {
      if (!item.enabled || !item.model) {
        continue;
      }
      const configID = item.id.trim();
      if (configID === "") {
        continue;
      }
      const modelID = item.model.model_id?.trim() ?? "";
      if (modelID === "") {
        continue;
      }
      const name = item.name?.trim() ?? "";
      const vendor = item.model.vendor?.trim() ?? "";
      const label = name || (vendor ? `${vendor} / ${modelID}` : modelID);
      map.set(configID, { label });
    }
    return map;
  });

  function resolveSemanticModelID(raw: string): string {
    return raw.trim();
  }

  function resolveModelLabel(modelConfigID: string): string {
    const normalized = modelConfigID.trim();
    if (normalized === "") {
      return "";
    }
    return enabledModelByConfigID.value.get(normalized)?.label ?? normalized;
  }

  const modelOptions = computed<Array<{ value: string; label: string }>>(() => {
    const project = input.activeProject.value;
    if (!project) {
      return [];
    }
    const projectConfig = projectStore.projectConfigsByProjectId[project.id];
    if (!projectConfig) {
      return [];
    }
    const configuredModelIDs = [
      ...(projectConfig.model_config_ids ?? []),
      projectConfig.default_model_config_id ?? ""
    ]
      .map((value) => value.trim())
      .filter((value, index, source) => value !== "" && source.indexOf(value) === index);

    return configuredModelIDs
      .filter((value) => enabledModelByConfigID.value.has(value))
      .map((value) => ({ value, label: resolveModelLabel(value) }));
  });

  const activeModelId = computed(() => {
    const runtimeModelID = resolveSemanticModelID(
      input.runtime.value?.modelId ?? input.activeConversation.value?.model_config_id ?? ""
    );
    if (runtimeModelID !== "" && modelOptions.value.some((item) => item.value === runtimeModelID)) {
      return runtimeModelID;
    }
    return modelOptions.value[0]?.value ?? "";
  });

  const activeModelLabel = computed(() => resolveModelLabel(activeModelId.value));

  return {
    modelOptions,
    activeModelId,
    activeModelLabel,
    resolveSemanticModelID
  };
}
