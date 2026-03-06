// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Maintainers
// SPDX-License-Identifier: MIT

import { computed, ref } from "vue";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";

import MainScreenView from "@/modules/session/views/MainScreenView.vue";
import { setLocale } from "@/shared/i18n";

const controllerState = vi.hoisted(() => ({
  current: null as ReturnType<typeof createControllerState> | null
}));

vi.mock("@/modules/session/views/useMainScreenController", () => ({
  useMainScreenController: () => controllerState.current
}));

describe("main screen view", () => {
  beforeEach(() => {
    setLocale("zh-CN");
    controllerState.current = createControllerState();
    vi.stubGlobal("matchMedia", vi.fn(() => ({
      matches: false,
      media: "(max-width: 768px)",
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn()
    })));
  });

  it("keeps the commit dialog open when commit fails", async () => {
    const wrapper = mount(MainScreenView, {
      global: {
        stubs: {
          MainShell: {
            template: `
              <div>
                <slot name="sidebar" />
                <slot name="header" />
                <slot name="main" />
                <slot name="footer" />
              </div>
            `
          },
          MainSidebarPanel: true,
          MainConversationPanel: true,
          MainInspectorPanel: true,
          AppIcon: true,
          HubStatusBar: true,
          Topbar: {
            template: `
              <div>
                <slot name="left" />
                <slot name="right" />
              </div>
            `
          }
        }
      }
    });

    const viewModel = wrapper.vm as unknown as {
      openCommitDialog: () => void;
      confirmCommitDialog: (message: string) => Promise<void>;
      commitDialogVisible: boolean;
    };

    viewModel.openCommitDialog();
    expect(viewModel.commitDialogVisible).toBe(true);

    await viewModel.confirmCommitDialog("chore: test inspector");

    expect(controllerState.current?.commitDiff).toHaveBeenCalledWith("chore: test inspector");
    expect(viewModel.commitDialogVisible).toBe(true);
  });
});

function createControllerState() {
  const activeConversation = createConversation();
  const activeProject = {
    id: "proj_1",
    name: "Inspector Project",
    is_git: true
  };
  const runtime = ref({
    draft: "",
    mode: "default",
    modelId: "rc_model_1",
    executions: [],
    events: [],
    changeSet: {
      change_set_id: "cs_1",
      conversation_id: activeConversation.id,
      project_kind: "git",
      file_count: 0,
      entries: [],
      capability: {
        can_commit: true,
        can_discard: true,
        can_export: true,
        can_export_patch: true
      },
      suggested_message: {
        message: "chore: test inspector"
      }
    },
    diffCapability: {
      can_commit: true,
      can_discard: true,
      can_export: true,
      can_export_patch: true
    },
    inspectorTab: "diff"
  });

  const commitDiff = vi.fn(async () => false);

  return {
    activeConversation: computed(() => activeConversation),
    activeConversationTokenUsage: computed(() => ({ input: 0, output: 0, total: 0 })),
    activeCount: computed(() => 0),
    activeProject: computed(() => activeProject),
    addConversationByPrompt: vi.fn(),
    approveExecution: vi.fn(),
    authStore: { me: { display_name: "local" } },
    changeInspectorTab: vi.fn(),
    clearComposerSuggestions: vi.fn(),
    commitDiff,
    composerSuggestions: ref([]),
    composerSuggesting: ref(false),
    changeRunTaskStateFilter: vi.fn(),
    controlRunTask: vi.fn(),
    conversationNameDraft: ref("Inspector Conversation"),
    conversationTokenUsageById: computed(() => ({})),
    conversationPageByProjectId: computed(() => ({})),
    createWorkspace: vi.fn(),
    deleteConversationById: vi.fn(),
    deleteProjectById: vi.fn(),
    denyExecution: vi.fn(),
    answerExecutionQuestion: vi.fn(),
    discardDiff: vi.fn(),
    editingConversationName: ref(false),
    executingCount: computed(() => 0),
    executionTraces: ref([]),
    exportConversation: vi.fn(),
    exportPatch: vi.fn(),
    importProjectDirectory: vi.fn(),
    isSwitchingModel: computed(() => false),
    inspectorCollapsed: ref(false),
    inspectorTabs: [
      { key: "diff", label: "D" },
      { key: "run", label: "R" },
      { key: "trace", label: "T" },
      { key: "risk", label: "!" }
    ],
    loginWorkspace: vi.fn(),
    nonGitCapability: {
      can_commit: false,
      can_discard: false,
      can_export: true,
      can_export_patch: true,
      reason: ""
    },
    onConversationNameInput: vi.fn(),
    openAccount: vi.fn(),
    openInspectorTab: vi.fn(),
    openSettings: vi.fn(),
    paginateConversations: vi.fn(),
    paginateProjects: vi.fn(),
    placeholder: computed(() => "输入消息"),
    pendingCount: computed(() => 0),
    pendingQuestions: ref([]),
    hasConfirmingExecution: computed(() => false),
    projectStore: {
      projects: [activeProject],
      conversationsByProjectId: {
        [activeProject.id]: [activeConversation]
      },
      activeConversationId: activeConversation.id
    },
    activeTraceCount: computed(() => 0),
    projectImportError: ref(""),
    projectImportFeedback: ref(""),
    projectImportInProgress: ref(false),
    projectsPage: computed(() => ({ canPrev: false, canNext: false, loading: false })),
    queuedCount: computed(() => 0),
    queuedMessages: ref([]),
    rollbackMessage: vi.fn(),
    renameConversation: vi.fn(),
    removeQueuedMessage: vi.fn(),
    runTaskDetailLoading: ref(false),
    runTaskListItems: ref([]),
    runTaskListLoading: ref(false),
    runTaskListNextCursor: ref(null),
    runTaskStateFilter: ref(""),
    runningState: computed(() => "stopped"),
    runningStateClass: computed(() => "stopped"),
    runningActions: ref([]),
    runtimeConnectionStatus: computed(() => "connected"),
    runtimeHubLabel: computed(() => "local://workspace"),
    runtimeUserDisplayName: computed(() => "local"),
    requestComposerSuggestions: vi.fn(),
    refreshRunTaskGraph: vi.fn(),
    selectTraceInInspector: vi.fn(),
    selectTraceMessage: vi.fn(),
    selectTraceExecution: vi.fn(),
    selectedTraceMessageId: ref(""),
    selectedTraceExecutionId: ref(""),
    selectedRunTask: ref(null),
    runTaskGraph: ref(null),
    runTaskGraphLoading: ref(false),
    runtime,
    loadMoreRunTasks: vi.fn(),
    selectRunTask: vi.fn(),
    visibleMessages: ref([]),
    saveConversationName: vi.fn(),
    selectConversation: vi.fn(),
    sendMessage: vi.fn(),
    startEditConversationName: vi.fn(),
    stopExecution: vi.fn(),
    switchWorkspace: vi.fn(),
    updateDraft: vi.fn(),
    activeModelLabel: computed(() => "gpt-5.3"),
    activeModelId: computed(() => "rc_model_1"),
    modelOptions: ref([{ value: "rc_model_1", label: "gpt-5.3" }]),
    updateMode: vi.fn(),
    updateModel: vi.fn(),
    workspaceLabel: computed(() => "本地工作区"),
    workspaceStore: {
      workspaces: [],
      currentWorkspaceId: "ws_local",
      connectionState: "ready",
      mode: "local"
    }
  };
}

function createConversation() {
  return {
    id: "conv_1",
    name: "Inspector Conversation",
    project_id: "proj_1",
    queue_state: "idle",
    default_mode: "default",
    model_config_id: "rc_model_1"
  };
}
