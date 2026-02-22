import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it } from "vitest";

import AppShell from "@/shared/layouts/AppShell.vue";
import { authStore, resetAuthStore } from "@/shared/stores/authStore";
import { resetWorkspaceStore, setCurrentWorkspace, setWorkspaces } from "@/shared/stores/workspaceStore";

describe("app shell", () => {
  beforeEach(() => {
    resetAuthStore();
    resetWorkspaceStore();
    setWorkspaces([
      {
        id: "ws_local",
        name: "Local Workspace",
        mode: "local",
        hub_url: null,
        is_default_local: true,
        created_at: "2026-02-22T00:00:00Z",
        login_disabled: false,
        auth_mode: "disabled"
      },
      {
        id: "ws_remote",
        name: "Remote Workspace",
        mode: "remote",
        hub_url: "https://hub.example.com",
        is_default_local: false,
        created_at: "2026-02-22T00:00:00Z",
        login_disabled: false,
        auth_mode: "password_or_token"
      }
    ]);
    setCurrentWorkspace("ws_local");
  });

  it("hides admin menu when admin capability is false", () => {
    authStore.capabilities = {
      admin_console: false,
      resource_write: false,
      execution_control: false
    };

    const wrapper = mountShell();
    expect(wrapper.text()).not.toContain("Admin");
  });

  it("shows admin menu when admin capability is true", () => {
    authStore.capabilities = {
      admin_console: true,
      resource_write: true,
      execution_control: true
    };

    const wrapper = mountShell();
    expect(wrapper.text()).toContain("Admin");
  });

  it("toggles settings panel and shows workspace/account info", async () => {
    authStore.me = {
      user_id: "u_001",
      display_name: "goya",
      workspace_id: "ws_local",
      role: "admin",
      capabilities: {
        admin_console: true,
        resource_write: true,
        execution_control: true
      }
    };

    const wrapper = mountShell();
    expect(wrapper.find('[data-testid="settings-panel"]').exists()).toBe(false);

    await wrapper.find('[data-testid="settings-toggle"]').trigger("click");
    expect(wrapper.find('[data-testid="settings-panel"]').exists()).toBe(true);
    expect(wrapper.text()).toContain("当前工作区账号信息");
    expect(wrapper.text()).toContain("Local Workspace");
    expect(wrapper.text()).toContain("goya");
  });

  it("renders workspace switcher options from workspace store", () => {
    const wrapper = mountShell();
    const options = wrapper.findAll("option");

    expect(options).toHaveLength(2);
    expect(options[0].text()).toContain("Local Workspace");
    expect(options[1].text()).toContain("Remote Workspace");
  });
});

function mountShell() {
  return mount(AppShell, {
    slots: {
      default: "<div>content</div>"
    },
    global: {
      stubs: {
        RouterLink: {
          template: "<a><slot /></a>"
        }
      }
    }
  });
}
