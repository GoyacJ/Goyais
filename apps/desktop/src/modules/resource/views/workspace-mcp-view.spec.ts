import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";

const mcpViewState = vi.hoisted(() => ({
  listItems: [] as Array<Record<string, unknown>>,
  connectSpy: vi.fn(),
  removeSpy: vi.fn(),
  confirmRemoveSpy: vi.fn(),
  closeRemoveSpy: vi.fn(),
  openCreateSpy: vi.fn(),
  openExportSpy: vi.fn()
}));

vi.mock("@/modules/resource/views/useWorkspaceMcpView", () => ({
  useWorkspaceMcpView: () => ({
    canWrite: true,
    closeExportModal: vi.fn(),
    closeModal: vi.fn(),
    connectStatusClass: vi.fn(() => "enabled"),
    connectSuggestion: vi.fn(() => ""),
    connect: mcpViewState.connectSpy,
    form: {
      open: false,
      mode: "create" as "create" | "edit",
      name: "",
      transport: "http_sse" as "http_sse" | "stdio",
      endpoint: "",
      command: "",
      envText: "",
      enabled: true,
      message: "",
      jsonModalOpen: false,
      removeModalOpen: false,
      removeConfigId: "",
      removeConfigName: ""
    },
    formatTime: vi.fn(() => "-"),
    getConnectResult: vi.fn(() => null),
    listState: {
      items: mcpViewState.listItems,
      page: {
        backStack: [] as Array<string | null>,
        nextCursor: null as string | null,
        loading: false
      },
      q: "",
      loading: false
    },
    loadNextPage: vi.fn(),
    loadPreviousPage: vi.fn(),
    onSearch: vi.fn(),
    openCreate: mcpViewState.openCreateSpy,
    openEdit: vi.fn(),
    openExportModal: mcpViewState.openExportSpy,
    applyExportPayload: vi.fn(),
    closeRemoveModal: mcpViewState.closeRemoveSpy,
    confirmRemoveConfig: mcpViewState.confirmRemoveSpy,
    removeConfig: mcpViewState.removeSpy,
    resourceStore: {
      error: "",
      mcpExport: null as Record<string, unknown> | null
    },
    saveConfig: vi.fn(),
    toggleEnabled: vi.fn()
  })
}));

import WorkspaceMcpView from "@/modules/resource/views/WorkspaceMcpView.vue";

describe("workspace mcp view", () => {
  it("renders empty state when no MCP configs", () => {
    mcpViewState.listItems = [];

    const wrapper = mount(WorkspaceMcpView, {
      global: {
        stubs: {
          WorkspaceSharedShell: {
            template: "<div class='workspace-shared-shell-stub'><slot /></div>"
          }
        }
      }
    });

    expect(wrapper.text()).toContain("暂无 MCP 配置");
    expect(wrapper.text()).toContain("单页");
  });

  it("triggers connect and remove actions from card buttons", async () => {
    mcpViewState.connectSpy.mockClear();
    mcpViewState.removeSpy.mockClear();
    mcpViewState.confirmRemoveSpy.mockClear();
    mcpViewState.listItems = [
      {
        id: "rc_mcp_click_1",
        name: "GitHub MCP",
        enabled: true,
        mcp: {
          transport: "http_sse",
          endpoint: "http://127.0.0.1:9001/sse",
          tools: ["repos.search"],
          last_connected_at: "2026-02-23T00:00:00Z"
        }
      }
    ];

    const wrapper = mount(WorkspaceMcpView, {
      global: {
        stubs: {
          WorkspaceSharedShell: {
            template: "<div class='workspace-shared-shell-stub'><slot /></div>"
          }
        }
      }
    });

    const connectButton = wrapper.findAll("button").find((item) => item.text() === "连接");
    const deleteButton = wrapper.findAll("button").find((item) => item.text() === "删除");

    expect(connectButton).toBeTruthy();
    expect(deleteButton).toBeTruthy();

    await connectButton?.trigger("click");
    await deleteButton?.trigger("click");

    expect(mcpViewState.connectSpy).toHaveBeenCalledTimes(1);
    expect(mcpViewState.removeSpy).toHaveBeenCalledTimes(1);
    expect(mcpViewState.confirmRemoveSpy).toHaveBeenCalledTimes(0);
  });
});
