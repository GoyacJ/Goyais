import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import MainSidebarPanel from "@/modules/session/components/MainSidebarPanel.vue";
import { pickDirectoryPath } from "@/shared/services/directoryPicker";
import { dismissToastByKey, showToast } from "@/shared/stores/toastStore";
import type { Project, Session } from "@/shared/types/api";

vi.mock("@/shared/services/directoryPicker", () => ({
  pickDirectoryPath: vi.fn()
}));

vi.mock("@/shared/stores/toastStore", () => ({
  showToast: vi.fn(),
  dismissToastByKey: vi.fn()
}));

describe("MainSidebarPanel", () => {
  beforeEach(() => {
    vi.mocked(pickDirectoryPath).mockReset();
    vi.mocked(showToast).mockReset();
    vi.mocked(dismissToastByKey).mockReset();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("emits importProject when plus button submits a directory path", async () => {
    vi.mocked(pickDirectoryPath).mockResolvedValue("/tmp/repo-alpha");
    const wrapper = mountSidebar();
    vi.mocked(showToast).mockClear();

    await wrapper.find(".projects-header .icon-btn").trigger("click");
    await flushPromises();

    expect(wrapper.emitted("importProject")).toEqual([["/tmp/repo-alpha"]]);
    expect(showToast).not.toHaveBeenCalled();
  });

  it("does not emit importProject when directory picker returns null and sends warning toast", async () => {
    vi.mocked(pickDirectoryPath).mockResolvedValue(null);
    const wrapper = mountSidebar();
    vi.mocked(showToast).mockClear();

    await wrapper.find(".projects-header .icon-btn").trigger("click");
    await flushPromises();

    expect(wrapper.emitted("importProject")).toBeUndefined();
    expect(showToast).toHaveBeenCalledWith(
      expect.objectContaining({
        tone: "warning",
        message: "未选择目录或目录读取失败，请重试"
      })
    );
  });

  it("disables plus button while project import is in progress and shows persistent toast", () => {
    const wrapper = mountSidebar({ projectImportInProgress: true });

    expect(wrapper.find(".projects-header .icon-btn").attributes("disabled")).toBeDefined();
    expect(showToast).toHaveBeenCalledWith(
      expect.objectContaining({
        key: "project-import-status",
        tone: "retrying",
        message: "正在导入项目...",
        persistent: true
      })
    );
  });

  it("maps project import feedback and error to keyed global toasts", async () => {
    const success = mountSidebar({ projectImportFeedback: "已添加项目：repo-alpha" });
    expect(showToast).toHaveBeenCalledWith(
      expect.objectContaining({
        key: "project-import-status",
        tone: "info",
        message: "已添加项目：repo-alpha"
      })
    );
    success.unmount();
    vi.mocked(showToast).mockClear();

    const failed = mountSidebar({ projectImportError: "ACCESS_DENIED: Permission denied (trace_id: tr_123)" });
    expect(showToast).toHaveBeenCalledWith(
      expect.objectContaining({
        key: "project-import-status",
        tone: "error",
        message: "ACCESS_DENIED: Permission denied (trace_id: tr_123)"
      })
    );
    failed.unmount();
  });

  it("shows auth required warning toast and keeps relogin button", () => {
    const wrapper = mountSidebar({ connectionState: "auth_required" });

    expect(wrapper.find(".auth-login-btn").exists()).toBe(true);
    expect(showToast).toHaveBeenCalledWith(
      expect.objectContaining({
        key: "workspace-auth-required",
        tone: "warning",
        message: "远程工作区鉴权已失效，请重新登录。",
        persistent: true
      })
    );
  });

  it("renders conversation token totals in list prefix", () => {
    const wrapper = mountSidebar({
      projects: [
        {
          id: "proj_1",
          workspace_id: "ws_local",
          name: "项目A",
          repo_path: "/tmp/project-a",
          is_git: true,
          token_threshold: 200000,
          tokens_total: 15603,
          current_revision: 0,
          created_at: "2026-02-23T00:00:00Z",
          updated_at: "2026-02-23T00:00:00Z"
        }
      ],
      conversationsByProjectId: {
        proj_1: [
          {
            id: "conv_1",
            workspace_id: "ws_local",
            project_id: "proj_1",
            name: "会话A",
            queue_state: "idle",
            default_mode: "default",
            model_config_id: "rc_model_1",
            rule_ids: [],
            skill_ids: [],
            mcp_ids: [],
            base_revision: 0,
            active_run_id: null,
            created_at: "2026-02-23T00:00:00Z",
            updated_at: "2026-02-23T00:00:00Z"
          }
        ]
      },
      conversationTokenUsageById: {
        conv_1: { input: 12, output: 20, total: 32 }
      }
    });

    expect(wrapper.find(".conversation-token").text()).toBe("32");
    expect(wrapper.find(".project-token").text()).toBe("15.6K / 200K");
    const projectTreeButton = wrapper.find(".tree-btn");
    const projectTreeSpans = projectTreeButton.findAll("span");
    expect(projectTreeSpans[0]?.text()).toBe("15.6K / 200K");
    expect(projectTreeSpans[1]?.text()).toBe("项目A");
  });

  it("starts inline rename on double click and emits rename on enter", async () => {
    const wrapper = mountSidebar({
      projects: [
        {
          id: "proj_1",
          workspace_id: "ws_local",
          name: "项目A",
          repo_path: "/tmp/project-a",
          is_git: true,
          current_revision: 0,
          created_at: "2026-02-23T00:00:00Z",
          updated_at: "2026-02-23T00:00:00Z"
        }
      ],
      conversationsByProjectId: {
        proj_1: [
          {
            id: "conv_1",
            workspace_id: "ws_local",
            project_id: "proj_1",
            name: "会话A",
            queue_state: "idle",
            default_mode: "default",
            model_config_id: "rc_model_1",
            rule_ids: [],
            skill_ids: [],
            mcp_ids: [],
            base_revision: 0,
            active_run_id: null,
            created_at: "2026-02-23T00:00:00Z",
            updated_at: "2026-02-23T00:00:00Z"
          }
        ]
      }
    });

    await wrapper.find(".conversation-main").trigger("dblclick");
    const input = wrapper.find(".conversation-rename-input");
    expect(input.exists()).toBe(true);

    await input.setValue("会话B");
    await input.trigger("keydown.enter");

    expect(wrapper.emitted("renameConversation")).toEqual([["proj_1", "conv_1", "会话B"]]);
    expect(wrapper.emitted("selectConversation")).toEqual([["proj_1", "conv_1"]]);
  });

  it("cancels inline rename on escape without emit", async () => {
    const wrapper = mountSidebar({
      projects: [
        {
          id: "proj_1",
          workspace_id: "ws_local",
          name: "项目A",
          repo_path: "/tmp/project-a",
          is_git: true,
          current_revision: 0,
          created_at: "2026-02-23T00:00:00Z",
          updated_at: "2026-02-23T00:00:00Z"
        }
      ],
      conversationsByProjectId: {
        proj_1: [
          {
            id: "conv_1",
            workspace_id: "ws_local",
            project_id: "proj_1",
            name: "会话A",
            queue_state: "idle",
            default_mode: "default",
            model_config_id: "rc_model_1",
            rule_ids: [],
            skill_ids: [],
            mcp_ids: [],
            base_revision: 0,
            active_run_id: null,
            created_at: "2026-02-23T00:00:00Z",
            updated_at: "2026-02-23T00:00:00Z"
          }
        ]
      }
    });

    await wrapper.find(".conversation-main").trigger("dblclick");
    const input = wrapper.find(".conversation-rename-input");
    await input.setValue("会话B");
    await input.trigger("keydown.esc");

    expect(wrapper.find(".conversation-rename-input").exists()).toBe(false);
    expect(wrapper.emitted("renameConversation")).toBeUndefined();
  });
});

function mountSidebar(
  overrides: Partial<{
    projectImportInProgress: boolean;
    projectImportFeedback: string;
    projectImportError: string;
    connectionState: string;
    projects: Project[];
    conversationsByProjectId: Record<string, Session[]>;
    conversationTokenUsageById: Record<string, { input: number; output: number; total: number }>;
  }> = {}
) {
  const now = "2026-02-23T00:00:00Z";
  return mount(MainSidebarPanel, {
    props: {
      workspaces: [
        {
          id: "ws_local",
          name: "Local Workspace",
          mode: "local",
          hub_url: null,
          is_default_local: true,
          created_at: now,
          login_disabled: true,
          auth_mode: "disabled"
        }
      ],
      currentWorkspaceId: "ws_local",
      workspaceMode: "local",
      workspaceName: "Local Workspace",
      userName: "local",
      projects: [],
      projectsPage: {
        canPrev: false,
        canNext: false,
        loading: false
      },
      conversationsByProjectId: {},
      conversationTokenUsageById: {},
      conversationPageByProjectId: {},
      activeConversationId: "",
      projectImportInProgress: false,
      projectImportFeedback: "",
      projectImportError: "",
      ...overrides
    },
    global: {
      stubs: {
        AppIcon: true,
        WorkspaceSwitcherCard: true,
        UserProfileMenuCard: true,
        WorkspaceCreateModal: true
      }
    }
  });
}
