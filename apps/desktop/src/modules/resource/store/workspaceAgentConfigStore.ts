import { computed } from "vue";
import { defineStore } from "pinia";

import { getWorkspaceAgentConfig, updateWorkspaceAgentConfig } from "@/modules/resource/services";
import { toDisplayError } from "@/shared/services/errorMapper";
import { pinia } from "@/shared/stores/pinia";
import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";
import type { TraceDetailLevel, WorkspaceAgentConfig } from "@/shared/types/api";

type WorkspaceAgentConfigState = {
  value: WorkspaceAgentConfig;
  initializedWorkspaceId: string;
  loading: boolean;
  saving: boolean;
  error: string;
};

const useWorkspaceAgentConfigStoreDefinition = defineStore("workspaceAgentConfig", {
  state: (): WorkspaceAgentConfigState => ({
    value: createDefaultWorkspaceAgentConfig(""),
    initializedWorkspaceId: "",
    loading: false,
    saving: false,
    error: ""
  })
});

export const useWorkspaceAgentConfigStateStore = useWorkspaceAgentConfigStoreDefinition;
const workspaceAgentConfigStore = useWorkspaceAgentConfigStoreDefinition(pinia);

export function useWorkspaceAgentConfigStore() {
  return {
    config: computed(() => workspaceAgentConfigStore.value),
    loading: computed(() => workspaceAgentConfigStore.loading),
    saving: computed(() => workspaceAgentConfigStore.saving),
    error: computed(() => workspaceAgentConfigStore.error),
    load: loadWorkspaceAgentConfig,
    update: updateWorkspaceAgentConfigPatch
  };
}

export async function loadWorkspaceAgentConfig(force = false): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }
  if (!force && workspaceAgentConfigStore.initializedWorkspaceId === workspace.id) {
    return;
  }

  workspaceAgentConfigStore.loading = true;
  workspaceAgentConfigStore.error = "";
  try {
    const loaded = await getWorkspaceAgentConfig(workspace.id);
    workspaceAgentConfigStore.value = normalizeWorkspaceAgentConfigForClient(loaded, workspace.id);
    workspaceAgentConfigStore.initializedWorkspaceId = workspace.id;
  } catch (error) {
    workspaceAgentConfigStore.value = createDefaultWorkspaceAgentConfig(workspace.id);
    workspaceAgentConfigStore.initializedWorkspaceId = workspace.id;
    workspaceAgentConfigStore.error = toDisplayError(error);
  } finally {
    workspaceAgentConfigStore.loading = false;
  }
}

type WorkspaceAgentConfigPatch = {
  max_model_turns?: number;
  show_process_trace?: boolean;
  trace_detail_level?: TraceDetailLevel;
};

export async function updateWorkspaceAgentConfigPatch(patch: WorkspaceAgentConfigPatch): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }
  if (workspaceAgentConfigStore.initializedWorkspaceId !== workspace.id) {
    await loadWorkspaceAgentConfig(true);
  }

  const next = normalizeWorkspaceAgentConfigForClient(
    {
      ...workspaceAgentConfigStore.value,
      workspace_id: workspace.id,
      execution: {
        max_model_turns: patch.max_model_turns ?? workspaceAgentConfigStore.value.execution.max_model_turns
      },
      display: {
        show_process_trace: patch.show_process_trace ?? workspaceAgentConfigStore.value.display.show_process_trace,
        trace_detail_level: patch.trace_detail_level ?? workspaceAgentConfigStore.value.display.trace_detail_level
      }
    },
    workspace.id
  );

  workspaceAgentConfigStore.value = next;
  workspaceAgentConfigStore.saving = true;
  workspaceAgentConfigStore.error = "";
  try {
    const saved = await updateWorkspaceAgentConfig(workspace.id, next);
    workspaceAgentConfigStore.value = normalizeWorkspaceAgentConfigForClient(saved, workspace.id);
    workspaceAgentConfigStore.initializedWorkspaceId = workspace.id;
  } catch (error) {
    workspaceAgentConfigStore.error = toDisplayError(error);
  } finally {
    workspaceAgentConfigStore.saving = false;
  }
}

export function resetWorkspaceAgentConfigStoreForTest(): void {
  workspaceAgentConfigStore.value = createDefaultWorkspaceAgentConfig("");
  workspaceAgentConfigStore.initializedWorkspaceId = "";
  workspaceAgentConfigStore.loading = false;
  workspaceAgentConfigStore.saving = false;
  workspaceAgentConfigStore.error = "";
}

function createDefaultWorkspaceAgentConfig(workspaceId: string): WorkspaceAgentConfig {
  return {
    workspace_id: workspaceId,
    execution: {
      max_model_turns: 24
    },
    display: {
      show_process_trace: true,
      trace_detail_level: "verbose"
    },
    updated_at: new Date().toISOString()
  };
}

function normalizeWorkspaceAgentConfigForClient(input: WorkspaceAgentConfig, workspaceId: string): WorkspaceAgentConfig {
  const maxTurnsCandidate = Number(input.execution?.max_model_turns ?? 24);
  const maxTurns = clampInteger(maxTurnsCandidate, 4, 64, 24);
  const traceDetail = input.display?.trace_detail_level === "basic" ? "basic" : "verbose";
  return {
    workspace_id: workspaceId || input.workspace_id || "",
    execution: {
      max_model_turns: maxTurns
    },
    display: {
      show_process_trace: input.display?.show_process_trace ?? true,
      trace_detail_level: traceDetail
    },
    updated_at: input.updated_at || new Date().toISOString()
  };
}

function clampInteger(value: number, min: number, max: number, fallback: number): number {
  if (!Number.isFinite(value)) {
    return fallback;
  }
  const normalized = Math.trunc(value);
  if (normalized < min) {
    return min;
  }
  if (normalized > max) {
    return max;
  }
  return normalized;
}
