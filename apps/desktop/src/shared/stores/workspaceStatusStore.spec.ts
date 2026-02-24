import { flushPromises, mount } from "@vue/test-utils";
import { computed, defineComponent, h } from "vue";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { resetAuthStore, setWorkspaceToken } from "@/shared/stores/authStore";
import { useWorkspaceStatusSync } from "@/shared/stores/workspaceStatusStore";
import { resetWorkspaceStore, setCurrentWorkspace, setWorkspaces } from "@/shared/stores/workspaceStore";
import type { WorkspaceStatusResponse } from "@/shared/types/api";

const getWorkspaceStatusMock = vi.fn();
const streamConversationEventsMock = vi.fn();

vi.mock("@/modules/workspace/services", () => ({
  getWorkspaceStatus: (...args: unknown[]) => getWorkspaceStatusMock(...args)
}));

vi.mock("@/modules/conversation/services", () => ({
  streamConversationEvents: (...args: unknown[]) => streamConversationEventsMock(...args)
}));

describe("workspace status sync", () => {
  beforeEach(() => {
    resetWorkspaceStore();
    resetAuthStore();
    getWorkspaceStatusMock.mockReset();
    streamConversationEventsMock.mockReset();
    vi.stubGlobal("EventSource", class {});
  });

  it("首拉成功并在 SSE 事件后刷新状态", async () => {
    prepareRemoteWorkspace();

    type StreamOptions = {
      onEvent: (event: unknown) => void;
      onStatusChange: (status: "connected" | "reconnecting" | "disconnected") => void;
      onError: (error: Error) => void;
    };
    let streamOptions: StreamOptions | undefined;
    const close = vi.fn();
    streamConversationEventsMock.mockImplementation((_conversationID: string, options: StreamOptions) => {
      streamOptions = options;
      return {
        close
      };
    });

    getWorkspaceStatusMock
      .mockResolvedValueOnce(buildStatusResponse({ conversation_status: "queued", conversation_id: "conv_sync" }))
      .mockResolvedValueOnce(buildStatusResponse({ conversation_status: "running", conversation_id: "conv_sync" }));

    const harness = mountHarness("conv_sync");
    await flushPromises();

    expect(getWorkspaceStatusMock).toHaveBeenNthCalledWith(1, "ws_remote", {
      conversationId: "conv_sync",
      token: "at_remote"
    });
    expect(streamConversationEventsMock).toHaveBeenCalledWith(
      "conv_sync",
      expect.objectContaining({
        token: "at_remote"
      })
    );
    expect(harness.api?.conversationStatus.value).toBe("queued");

    streamOptions?.onEvent({ event_id: "evt_1" });
    await flushPromises();

    expect(getWorkspaceStatusMock).toHaveBeenCalledTimes(2);
    expect(harness.api?.conversationStatus.value).toBe("running");

    harness.wrapper.unmount();
    expect(close).toHaveBeenCalled();
  });

  it("切换 activeConversation 时会重绑 SSE", async () => {
    prepareRemoteWorkspace();

    const closeFirst = vi.fn();
    const closeSecond = vi.fn();
    streamConversationEventsMock
      .mockImplementationOnce(() => ({
        close: closeFirst
      }))
      .mockImplementationOnce(() => ({
        close: closeSecond
      }));

    getWorkspaceStatusMock
      .mockResolvedValueOnce(buildStatusResponse({ conversation_id: "conv_a", conversation_status: "queued" }))
      .mockResolvedValueOnce(buildStatusResponse({ conversation_id: "conv_b", conversation_status: "done" }));

    const harness = mountHarness("conv_a");
    await flushPromises();

    await harness.wrapper.setProps({ conversationId: "conv_b" });
    await flushPromises();

    expect(getWorkspaceStatusMock).toHaveBeenCalledWith("ws_remote", {
      conversationId: "conv_b",
      token: "at_remote"
    });
    expect(closeFirst).toHaveBeenCalledTimes(1);
    expect(streamConversationEventsMock).toHaveBeenNthCalledWith(2, "conv_b", expect.any(Object));
    expect(harness.api?.conversationStatus.value).toBe("done");

    harness.wrapper.unmount();
    expect(closeSecond).toHaveBeenCalled();
  });

  it("接口失败时降级为 stopped/disconnected 且不建立 SSE", async () => {
    prepareRemoteWorkspace();
    getWorkspaceStatusMock.mockRejectedValueOnce(new Error("status unavailable"));

    const harness = mountHarness("conv_error");
    await flushPromises();

    expect(streamConversationEventsMock).not.toHaveBeenCalled();
    expect(harness.api?.conversationStatus.value).toBe("stopped");
    expect(harness.api?.connectionStatus.value).toBe("disconnected");
    expect(harness.api?.error.value).toContain("status unavailable");

    harness.wrapper.unmount();
  });
});

function prepareRemoteWorkspace(): void {
  setWorkspaces([
    {
      id: "ws_local",
      name: "Local Workspace",
      mode: "local",
      hub_url: null,
      is_default_local: true,
      created_at: "2026-02-23T00:00:00Z",
      login_disabled: true,
      auth_mode: "disabled"
    },
    {
      id: "ws_remote",
      name: "Remote Workspace",
      mode: "remote",
      hub_url: "https://hub.example.com",
      is_default_local: false,
      created_at: "2026-02-24T00:00:00Z",
      login_disabled: false,
      auth_mode: "password_or_token"
    }
  ]);
  setCurrentWorkspace("ws_remote");
  setWorkspaceToken("ws_remote", "at_remote");
}

function mountHarness(conversationId: string) {
  const apiRef: { current: ReturnType<typeof useWorkspaceStatusSync> | undefined } = {
    current: undefined
  };
  const Harness = defineComponent({
    props: {
      conversationId: {
        type: String,
        default: ""
      }
    },
    setup(props) {
      apiRef.current = useWorkspaceStatusSync({
        conversationId: computed(() => props.conversationId)
      });
      return () => h("div");
    }
  });

  const wrapper = mount(Harness, {
    props: {
      conversationId
    }
  });

  return {
    wrapper,
    get api() {
      return apiRef.current;
    }
  };
}

function buildStatusResponse(partial: Partial<WorkspaceStatusResponse>): WorkspaceStatusResponse {
  return {
    workspace_id: "ws_remote",
    conversation_id: "conv_sync",
    conversation_status: "stopped",
    hub_url: "https://hub.example.com",
    connection_status: "connected",
    user_display_name: "Remote User",
    updated_at: "2026-02-24T00:00:00Z",
    ...partial
  };
}
