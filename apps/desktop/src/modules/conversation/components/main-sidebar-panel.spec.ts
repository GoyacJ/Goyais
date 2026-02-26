import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import MainSidebarPanel from "@/modules/conversation/components/MainSidebarPanel.vue";
import { pickDirectoryPath } from "@/shared/services/directoryPicker";
import { dismissToastByKey, showToast } from "@/shared/stores/toastStore";

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
});

function mountSidebar(
  overrides: Partial<{
    projectImportInProgress: boolean;
    projectImportFeedback: string;
    projectImportError: string;
    connectionState: string;
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
