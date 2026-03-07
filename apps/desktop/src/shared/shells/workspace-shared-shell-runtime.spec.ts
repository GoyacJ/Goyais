import { computed } from "vue";
import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import WorkspaceSharedShell from "@/shared/shells/WorkspaceSharedShell.vue";
import { resetWorkspaceStore, setCurrentWorkspace, setWorkspaces } from "@/shared/stores/workspaceStore";

vi.mock("@/shared/layouts/useConfigRuntimeStatus", () => {
  return {
    useConfigRuntimeStatus: () => ({
      runtimeStatusMode: computed(() => true),
      conversationStatus: computed(() => "running"),
      connectionStatus: computed(() => "connected"),
      userDisplayName: computed(() => "Mock User"),
      hubUrl: computed(() => "local://workspace"),
      conversationLabel: computed(() => "conversation: running"),
      connectionLabel: computed(() => "connected"),
      connectionTone: computed(() => "connected")
    })
  };
});

describe("workspace shared shell runtime wiring", () => {
  beforeEach(() => {
    resetWorkspaceStore();
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
  });

  it("passes runtime props to SettingsShell in local mode", () => {
    setCurrentWorkspace("ws_local");

    const wrapper = mount(WorkspaceSharedShell, {
      props: {
        activeKey: "workspace_rules",
        title: "规则配置",
        accountSubtitle: "Workspace Config / Rules",
        settingsSubtitle: "Local Settings / Rules"
      },
      global: {
        stubs: {
          SettingsShell: {
            props: ["runtimeStatusMode", "runtimeConversationStatus", "runtimeConnectionStatus", "runtimeUserDisplayName", "runtimeHubUrl"],
            template:
              '<div class="settings-shell-stub">{{ runtimeStatusMode }}|{{ runtimeConversationStatus }}|{{ runtimeConnectionStatus }}|{{ runtimeUserDisplayName }}|{{ runtimeHubUrl }}<slot /></div>'
          },
          AccountShell: {
            template: '<div class="account-shell-stub"><slot /></div>'
          }
        }
      }
    });

    expect(wrapper.find(".settings-shell-stub").exists()).toBe(true);
    expect(wrapper.find(".settings-shell-stub").text()).toContain("true|running|connected|Mock User|local://workspace");
    expect(wrapper.find(".account-shell-stub").exists()).toBe(false);
  });

  it("passes runtime props to AccountShell in remote mode", () => {
    setCurrentWorkspace("ws_remote");

    const wrapper = mount(WorkspaceSharedShell, {
      props: {
        activeKey: "workspace_rules",
        title: "规则配置",
        accountSubtitle: "Workspace Config / Rules",
        settingsSubtitle: "Local Settings / Rules"
      },
      global: {
        stubs: {
          AccountShell: {
            props: ["runtimeStatusMode", "runtimeConversationStatus", "runtimeConnectionStatus", "runtimeUserDisplayName", "runtimeHubUrl"],
            template:
              '<div class="account-shell-stub">{{ runtimeStatusMode }}|{{ runtimeConversationStatus }}|{{ runtimeConnectionStatus }}|{{ runtimeUserDisplayName }}|{{ runtimeHubUrl }}<slot /></div>'
          },
          SettingsShell: {
            template: '<div class="settings-shell-stub"><slot /></div>'
          }
        }
      }
    });

    expect(wrapper.find(".account-shell-stub").exists()).toBe(true);
    expect(wrapper.find(".account-shell-stub").text()).toContain("true|running|connected|Mock User|local://workspace");
    expect(wrapper.find(".settings-shell-stub").exists()).toBe(false);
  });
});
