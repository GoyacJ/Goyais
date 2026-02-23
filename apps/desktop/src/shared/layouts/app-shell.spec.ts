import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";

import type { MenuEntry } from "@/shared/navigation/pageMenus";
import RemoteConfigSidebar from "@/shared/layouts/RemoteConfigSidebar.vue";
import { resetAuthStore } from "@/shared/stores/authStore";
import { resetWorkspaceStore, setCurrentWorkspace, setWorkspaces } from "@/shared/stores/workspaceStore";

const menuEntries: MenuEntry[] = [
  { key: "remote_account", label: "账号信息", path: "/remote/account", visibility: "enabled" },
  { key: "remote_members_roles", label: "成员与角色", path: "/remote/members-roles", visibility: "enabled" },
  { key: "remote_permissions_audit", label: "权限与审计", path: "/remote/permissions-audit", visibility: "enabled" }
];

describe("remote config sidebar", () => {
  beforeEach(() => {
    resetAuthStore();
    resetWorkspaceStore();
    setWorkspaces([
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
  });

  it("renders menu and active item", () => {
    const wrapper = mountSidebar();
    expect(wrapper.text()).toContain("账号信息");
    expect(wrapper.find(".menu-item.active").text()).toContain("账号信息");
  });

  it("toggles workspace menu", async () => {
    const wrapper = mountSidebar();
    expect(wrapper.find(".workspace-menu").exists()).toBe(false);

    await wrapper.find(".workspace-btn").trigger("click");
    expect(wrapper.find(".workspace-menu").exists()).toBe(true);
  });

  it("toggles user menu", async () => {
    const wrapper = mountSidebar();
    expect(wrapper.find(".user-menu").exists()).toBe(false);

    await wrapper.find(".user-trigger").trigger("click");
    expect(wrapper.find(".user-menu").exists()).toBe(true);
    expect(wrapper.text()).toContain("设置");
  });
});

function mountSidebar() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:pathMatch(.*)*", component: { template: "<div />" } }]
  });

  return mount(RemoteConfigSidebar, {
    props: {
      activeKey: "remote_account",
      scopeHint: "Remote 视图提示",
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
