import { computed, reactive } from "vue";

import { getWorkspaceAgentConfig, updateWorkspaceAgentConfig } from "@/modules/resource/services";
import { toDisplayError } from "@/shared/services/errorMapper";
import { getCurrentWorkspace } from "@/shared/stores/workspaceStore";
import type { TraceDetailLevel, WorkspaceAgentConfig } from "@/shared/types/api";

type WorkspaceAgentConfigState = {
  value: WorkspaceAgentConfig;
  initializedWorkspaceId: string;
  loading: boolean;
  saving: boolean;
  error: string;
};

const state = reactive<WorkspaceAgentConfigState>({
  value: createDefaultWorkspaceAgentConfig(""),
  initializedWorkspaceId: "",
  loading: false,
  saving: false,
  error: ""
});

export function useWorkspaceAgentConfigStore() {
  return {
    config: computed(() => state.value),
    loading: computed(() => state.loading),
    saving: computed(() => state.saving),
    error: computed(() => state.error),
    load: loadWorkspaceAgentConfig,
    update: updateWorkspaceAgentConfigPatch
  };
}

export async function loadWorkspaceAgentConfig(force = false): Promise<void> {
  const workspace = getCurrentWorkspace();
  if (!workspace) {
    return;
  }
  if (!force && state.initializedWorkspaceId === workspace.id) {
    return;
  }

  state.loading = true;
  state.error = "";
  try {
    const loaded = await getWorkspaceAgentConfig(workspace.id);
    state.value = normalizeWorkspaceAgentConfigForClient(loaded, workspace.id);
    state.initializedWorkspaceId = workspace.id;
  } catch (error) {
    state.value = createDefaultWorkspaceAgentConfig(workspace.id);
    state.initializedWorkspaceId = workspace.id;
    state.error = toDisplayError(error);
  } finally {
    state.loading = false;
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
  if (state.initializedWorkspaceId !== workspace.id) {
    await loadWorkspaceAgentConfig(true);
  }

  const next = normalizeWorkspaceAgentConfigForClient(
    {
      ...state.value,
      workspace_id: workspace.id,
      execution: {
        max_model_turns: patch.max_model_turns ?? state.value.execution.max_model_turns
      },
      display: {
        show_process_trace: patch.show_process_trace ?? state.value.display.show_process_trace,
        trace_detail_level: patch.trace_detail_level ?? state.value.display.trace_detail_level
      }
    },
    workspace.id
  );

  state.value = next;
  state.saving = true;
  state.error = "";
  try {
    const saved = await updateWorkspaceAgentConfig(workspace.id, next);
    state.value = normalizeWorkspaceAgentConfigForClient(saved, workspace.id);
    state.initializedWorkspaceId = workspace.id;
  } catch (error) {
    state.error = toDisplayError(error);
  } finally {
    state.saving = false;
  }
}

export function resetWorkspaceAgentConfigStoreForTest(): void {
  state.value = createDefaultWorkspaceAgentConfig("");
  state.initializedWorkspaceId = "";
  state.loading = false;
  state.saving = false;
  state.error = "";
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
