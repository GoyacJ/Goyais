import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import MainSidebarPanel from "@/modules/conversation/components/MainSidebarPanel.vue";
import { pickDirectoryPath } from "@/shared/services/directoryPicker";

vi.mock("@/shared/services/directoryPicker", () => ({
  pickDirectoryPath: vi.fn()
}));

describe("MainSidebarPanel", () => {
  beforeEach(() => {
    vi.mocked(pickDirectoryPath).mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("emits importProject when plus button submits a directory path", async () => {
    vi.mocked(pickDirectoryPath).mockResolvedValue("/tmp/repo-alpha");
    const wrapper = mountSidebar();

    await wrapper.find(".projects-header .icon-btn").trigger("click");
    await flushPromises();

    expect(wrapper.emitted("importProject")).toEqual([["/tmp/repo-alpha"]]);
  });

  it("does not emit importProject when directory picker returns null", async () => {
    vi.mocked(pickDirectoryPath).mockResolvedValue(null);
    const wrapper = mountSidebar();

    await wrapper.find(".projects-header .icon-btn").trigger("click");
    await flushPromises();

    expect(wrapper.emitted("importProject")).toBeUndefined();
    expect(wrapper.text()).toContain("未选择目录或目录读取失败");
  });

  it("disables plus button while project import is in progress", () => {
    const wrapper = mountSidebar({ projectImportInProgress: true });
    expect(wrapper.find(".projects-header .icon-btn").attributes("disabled")).toBeDefined();
    expect(wrapper.text()).toContain("正在导入项目");
  });

  it("renders import success and error feedback", async () => {
    const success = mountSidebar({ projectImportFeedback: "已添加项目：repo-alpha" });
    expect(success.text()).toContain("已添加项目：repo-alpha");

    const failed = mountSidebar({ projectImportError: "ACCESS_DENIED: Permission denied (trace_id: tr_123)" });
    expect(failed.text()).toContain("ACCESS_DENIED");
    expect(failed.text()).toContain("trace_id: tr_123");
  });
});

function mountSidebar(
  overrides: Partial<{
    projectImportInProgress: boolean;
    projectImportFeedback: string;
    projectImportError: string;
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
        WorkspaceCreateModal: true,
        ToastAlert: {
          props: ["message"],
          template: "<div class='toast-stub'>{{ message }}</div>"
        }
      }
    }
  });
}
