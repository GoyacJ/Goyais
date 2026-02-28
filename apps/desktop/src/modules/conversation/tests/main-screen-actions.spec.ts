import { computed, ref } from "vue";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Router } from "vue-router";

import type { ConversationRuntime } from "@/modules/conversation/store/state";
import { useMainScreenActions } from "@/modules/conversation/views/useMainScreenActions";
import type { Conversation, ConversationMessage, Project } from "@/shared/types/api";

const conversationStoreMocks = vi.hoisted(() => ({
  approveConversationExecution: vi.fn(),
  commitLatestDiff: vi.fn(),
  denyConversationExecution: vi.fn(),
  discardLatestDiff: vi.fn(),
  getLatestFinishedExecution: vi.fn(),
  rollbackConversationToMessage: vi.fn(),
  setConversationDraft: vi.fn(),
  setConversationError: vi.fn(),
  setConversationInspectorTab: vi.fn(),
  setConversationMode: vi.fn(),
  setConversationModel: vi.fn(),
  stopConversationExecution: vi.fn(),
  submitConversationMessage: vi.fn()
}));

const projectStoreMocks = vi.hoisted(() => ({
  addConversation: vi.fn(),
  deleteConversation: vi.fn(),
  deleteProject: vi.fn(),
  exportConversationById: vi.fn(),
  importProjectByDirectory: vi.fn(),
  loadNextConversationsPage: vi.fn(),
  loadNextProjectsPage: vi.fn(),
  loadPreviousConversationsPage: vi.fn(),
  loadPreviousProjectsPage: vi.fn(),
  renameConversationById: vi.fn(),
  setActiveConversation: vi.fn(),
  setActiveProject: vi.fn(),
  updateConversationModeById: vi.fn(),
  updateConversationModelById: vi.fn(),
  refreshProjects: vi.fn(),
  projectStore: {
    projects: [] as Project[],
    conversationsByProjectId: {} as Record<string, Conversation[]>,
    error: ""
  }
}));

const conversationServiceMocks = vi.hoisted(() => ({
  exportExecutionPatch: vi.fn()
}));

const workspaceServiceMocks = vi.hoisted(() => ({
  createRemoteConnection: vi.fn(),
  loginWorkspace: vi.fn()
}));

const authStoreMocks = vi.hoisted(() => ({
  refreshMeForCurrentWorkspace: vi.fn(),
  setWorkspaceToken: vi.fn()
}));

const workspaceStoreMocks = vi.hoisted(() => ({
  workspaceStore: {
    currentWorkspaceId: "",
    connectionState: "ready",
    mode: "local"
  },
  setWorkspaceConnection: vi.fn(),
  switchWorkspaceContext: vi.fn(),
  upsertWorkspace: vi.fn()
}));

vi.mock("@/modules/conversation/store", () => conversationStoreMocks);
vi.mock("@/modules/project/store", () => projectStoreMocks);
vi.mock("@/modules/conversation/services", () => conversationServiceMocks);
vi.mock("@/modules/admin/store", () => ({
  refreshAdminData: vi.fn()
}));
vi.mock("@/modules/resource/store", () => ({
  refreshResources: vi.fn(),
  refreshModelCatalog: vi.fn()
}));
vi.mock("@/modules/workspace/services", () => workspaceServiceMocks);
vi.mock("@/shared/stores/authStore", () => authStoreMocks);
vi.mock("@/shared/services/errorMapper", () => ({
  toDisplayError: (error: unknown) => (error instanceof Error ? error.message : "unknown")
}));
vi.mock("@/shared/stores/workspaceStore", () => ({
  workspaceStore: workspaceStoreMocks.workspaceStore
}));
vi.mock("@/modules/workspace/store", () => ({
  setWorkspaceConnection: workspaceStoreMocks.setWorkspaceConnection,
  switchWorkspaceContext: workspaceStoreMocks.switchWorkspaceContext,
  upsertWorkspace: workspaceStoreMocks.upsertWorkspace
}));

describe("main screen actions - auto conversation naming", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    projectStoreMocks.projectStore.projects = [];
    projectStoreMocks.projectStore.conversationsByProjectId = {};
    projectStoreMocks.projectStore.error = "";
  });

  it("renames default conversation from first user message after submit", async () => {
    conversationStoreMocks.submitConversationMessage.mockResolvedValue(undefined);
    const { actions, project, conversation } = createActionsContext({
      conversationName: "新对话 2",
      draft: "  这是第一行\n第二行还有  "
    });

    await actions.sendMessage();

    expect(conversationStoreMocks.submitConversationMessage).toHaveBeenCalledWith(conversation, true, {
      catalogRevision: "rev_test"
    });
    expect(projectStoreMocks.renameConversationById).toHaveBeenCalledWith(project.id, conversation.id, "这是第一行 第二行还");

    const submitOrder = conversationStoreMocks.submitConversationMessage.mock.invocationCallOrder[0] ?? 0;
    const renameOrder = projectStoreMocks.renameConversationById.mock.invocationCallOrder[0] ?? 0;
    expect(submitOrder).toBeGreaterThan(0);
    expect(renameOrder).toBeGreaterThan(submitOrder);
  });

  it("does not rename when current name is custom", async () => {
    conversationStoreMocks.submitConversationMessage.mockResolvedValue(undefined);
    const { actions, conversation } = createActionsContext({
      conversationName: "我的自定义会话",
      draft: "这是首条消息"
    });

    await actions.sendMessage();

    expect(conversationStoreMocks.submitConversationMessage).toHaveBeenCalledWith(conversation, true, {
      catalogRevision: "rev_test"
    });
    expect(projectStoreMocks.renameConversationById).not.toHaveBeenCalled();
  });

  it("does not rename when a user message already exists", async () => {
    conversationStoreMocks.submitConversationMessage.mockResolvedValue(undefined);
    const existingUserMessage: ConversationMessage = {
      id: "msg_existing",
      conversation_id: "conv_1",
      role: "user",
      content: "old",
      created_at: "2026-02-26T00:00:00Z"
    };
    const { actions } = createActionsContext({
      conversationName: "新对话 1",
      draft: "这是第二条消息",
      runtimeMessages: [existingUserMessage]
    });

    await actions.sendMessage();

    expect(projectStoreMocks.renameConversationById).not.toHaveBeenCalled();
  });

  it("renames even when submit message throws", async () => {
    conversationStoreMocks.submitConversationMessage.mockRejectedValue(new Error("submit failed"));
    const { actions, project, conversation } = createActionsContext({
      conversationName: "Conversation",
      draft: "first message for title"
    });

    await expect(actions.sendMessage()).resolves.toBeUndefined();

    expect(projectStoreMocks.renameConversationById).toHaveBeenCalledWith(project.id, conversation.id, "first mess");
  });

  it("keeps manual rename editing flow available", async () => {
    const { actions, inputRefs, project, conversation } = createActionsContext({
      conversationName: "已有名称",
      draft: "ignored"
    });

    actions.startEditConversationName();
    expect(inputRefs.editingConversationName.value).toBe(true);
    expect(inputRefs.conversationNameDraft.value).toBe("已有名称");

    inputRefs.conversationNameDraft.value = "手动改名";
    await actions.saveConversationName();

    expect(projectStoreMocks.renameConversationById).toHaveBeenCalledWith(project.id, conversation.id, "手动改名");
  });
});

function createActionsContext(input: {
  conversationName: string;
  draft: string;
  runtimeMessages?: ConversationMessage[];
}) {
  const conversation = createConversation(input.conversationName);
  const project = createProject();
  const runtimeValue = createRuntime({
    draft: input.draft,
    messages: input.runtimeMessages ?? []
  });

  const activeConversationRef = ref<Conversation | undefined>(conversation);
  const activeProjectRef = ref<Project | undefined>(project);
  const runtimeRef = ref<ConversationRuntime | undefined>(runtimeValue);

  const inputRefs = {
    inspectorCollapsed: ref(false),
    editingConversationName: ref(false),
    conversationNameDraft: ref(""),
    projectImportInProgress: ref(false),
    projectImportFeedback: ref(""),
    projectImportError: ref("")
  };

  const actions = useMainScreenActions({
    router: { push: vi.fn() } as unknown as Router,
    activeConversation: computed(() => activeConversationRef.value),
    activeProject: computed(() => activeProjectRef.value),
    runtime: computed(() => runtimeRef.value),
    modelOptions: computed(() => [{ value: "rc_model_1", label: "rc_model_1" }]),
    composerCatalogRevision: computed(() => "rev_test"),
    inspectorCollapsed: inputRefs.inspectorCollapsed,
    editingConversationName: inputRefs.editingConversationName,
    conversationNameDraft: inputRefs.conversationNameDraft,
    projectImportInProgress: inputRefs.projectImportInProgress,
    projectImportFeedback: inputRefs.projectImportFeedback,
    projectImportError: inputRefs.projectImportError,
    resolveSemanticModelID: (raw) => raw
  });

  return {
    actions,
    project,
    conversation,
    inputRefs
  };
}

function createConversation(name: string): Conversation {
  return {
    id: "conv_1",
    workspace_id: "ws_1",
    project_id: "proj_1",
    name,
    queue_state: "idle",
    default_mode: "agent",
    model_config_id: "rc_model_1",
    rule_ids: [],
    skill_ids: [],
    mcp_ids: [],
    base_revision: 0,
    active_execution_id: null,
    created_at: "2026-02-26T00:00:00Z",
    updated_at: "2026-02-26T00:00:00Z"
  };
}

function createProject(): Project {
  return {
    id: "proj_1",
    workspace_id: "ws_1",
    name: "proj",
    repo_path: "/tmp/proj",
    is_git: true,
    default_model_config_id: "rc_model_1",
    default_mode: "agent",
    current_revision: 0,
    created_at: "2026-02-26T00:00:00Z",
    updated_at: "2026-02-26T00:00:00Z"
  };
}

function createRuntime(overrides: Partial<ConversationRuntime> = {}): ConversationRuntime {
  return {
    messages: [],
    events: [],
    executions: [],
    snapshots: [],
    draft: "",
    mode: "agent",
    modelId: "rc_model_1",
    ruleIds: [],
    skillIds: [],
    mcpIds: [],
    status: "connected",
    diff: [],
    diffCapability: {
      can_commit: true,
      can_discard: true,
      can_export_patch: true
    },
    inspectorTab: "diff",
    worktreeRef: null,
    hydrated: false,
    lastEventId: "",
    processedEventKeys: [],
    processedEventKeySet: new Set<string>(),
    completionMessageKeys: [],
    completionMessageKeySet: new Set<string>(),
    ...overrides
  };
}
