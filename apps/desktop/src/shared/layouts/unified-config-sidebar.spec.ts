import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";

import type { MenuEntry } from "@/shared/navigation/pageMenus";
import UnifiedConfigSidebar from "@/shared/layouts/UnifiedConfigSidebar.vue";
import { authStore, resetAuthStore } from "@/shared/stores/authStore";
import { resetWorkspaceStore, setCurrentWorkspace, setWorkspaces } from "@/shared/stores/workspaceStore";

const menuEntries: MenuEntry[] = [
  { key: "remote_account", label: "账号信息", path: "/remote/account", visibility: "enabled" },
  { key: "workspace_agent", label: "Agent配置", path: "/workspace/agent", visibility: "enabled" },
  { key: "settings_theme", label: "主题", path: "/settings/theme", visibility: "enabled" }
];

describe("unified config sidebar", () => {
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
        created_at: "2026-02-23T00:00:00Z",
        login_disabled: false,
        auth_mode: "password_or_token"
      }
    ]);
    setCurrentWorkspace("ws_remote");
    authStore.me = {
      user_id: "u_admin",
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

  it("shows remote groups when variant is remote", () => {
    const wrapper = mountSidebar("remote");
    expect(wrapper.text()).toContain("远程管理");
    expect(wrapper.text()).toContain("工作区配置");
  });

  it("shows settings groups when variant is local", () => {
    const wrapper = mountSidebar("local");
    expect(wrapper.text()).toContain("软件通用设置");
    expect(wrapper.text()).not.toContain("远程管理");
  });
});

function mountSidebar(variant: "local" | "remote") {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:pathMatch(.*)*", component: { template: "<div />" } }]
  });

  return mount(UnifiedConfigSidebar, {
    props: {
      variant,
      activeKey: variant === "remote" ? "remote_account" : "settings_theme",
      menuEntries
    },
    global: {
      plugins: [router],
      stubs: {
        RouterLink: {
          props: ["to"],
          template: "<a><slot /></a>"
        }
      }
    }
  });
}
