import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it } from "vitest";

import { setLocale } from "@/shared/i18n";
import { authStore, resetAuthStore } from "@/shared/stores/authStore";
import { resetWorkspaceStore, setCurrentWorkspace, setWorkspaces } from "@/shared/stores/workspaceStore";
import HubStatusBar from "@/shared/ui/HubStatusBar.vue";

describe("hub status bar", () => {
  beforeEach(() => {
    resetWorkspaceStore();
    resetAuthStore();
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
    setLocale("en-US");
    authStore.me = {
      user_id: "u_remote",
      display_name: "Remote Admin",
      workspace_id: "ws_remote",
      role: "admin",
      capabilities: {
        admin_console: true,
        resource_write: true,
        execution_control: true
      }
    };
  });

  it("runtime 模式显示用户名与连接状态", () => {
    const wrapper = mount(HubStatusBar, {
      props: {
        runtimeMode: true,
        hubLabel: "https://hub.example.com",
        userLabel: "Alice",
        connectionStatus: "connected"
      }
    });

    expect(wrapper.text()).toContain("Hub: https://hub.example.com");
    expect(wrapper.text()).toContain("Alice");
    expect(wrapper.text()).toContain("connected");
    expect(wrapper.text()).not.toContain("admin");
  });

  it("默认模式沿用角色显示", () => {
    const wrapper = mount(HubStatusBar, {
      props: {
        roleLabel: "owner",
        connectionState: "loading"
      }
    });

    expect(wrapper.text()).toContain("owner");
    expect(wrapper.text()).toContain("reconnecting");
  });

  it("ignores empty runtime labels and falls back to workspace/auth state", () => {
    const wrapper = mount(HubStatusBar, {
      props: {
        runtimeMode: true,
        hubLabel: "   ",
        userLabel: " ",
        connectionStatus: undefined
      }
    });

    expect(wrapper.text()).toContain("Hub: https://hub.example.com");
    expect(wrapper.text()).toContain("Remote Admin");
    expect(wrapper.text()).toContain("disconnected");
  });

  it("localizes runtime labels by current locale", () => {
    setLocale("zh-CN");

    const wrapper = mount(HubStatusBar, {
      props: {
        runtimeMode: true,
        hubLabel: "local://workspace",
        userLabel: "本地测试用户",
        connectionStatus: "connected"
      }
    });

    expect(wrapper.text()).toContain("Hub: local://workspace");
    expect(wrapper.text()).toContain("本地测试用户");
    expect(wrapper.text()).toContain("已连接");
  });
});
