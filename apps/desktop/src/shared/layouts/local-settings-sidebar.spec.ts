import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";

import type { MenuEntry } from "@/shared/navigation/pageMenus";
import LocalSettingsSidebar from "@/shared/layouts/LocalSettingsSidebar.vue";
import { resetWorkspaceStore, setCurrentWorkspace, setWorkspaces } from "@/shared/stores/workspaceStore";

const menuEntries: MenuEntry[] = [
  { key: "workspace_agent", label: "Agent配置", path: "/workspace/agent", visibility: "enabled" },
  { key: "settings_theme", label: "主题", path: "/settings/theme", visibility: "enabled" }
];

describe("local settings sidebar", () => {
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
        created_at: "2026-02-23T00:00:00Z",
        login_disabled: false,
        auth_mode: "password_or_token"
      }
    ]);
    setCurrentWorkspace("ws_local");
  });

  it("shows add workspace entry in workspace menu", async () => {
    const wrapper = mountSidebar();
    await wrapper.find(".workspace-btn").trigger("click");
    expect(wrapper.find(".workspace-menu").exists()).toBe(true);
    expect(wrapper.text()).toContain("新增工作区");
  });

  it("opens create workspace modal when selecting add workspace", async () => {
    const wrapper = mountSidebar();
    await wrapper.find(".workspace-btn").trigger("click");
    await wrapper.find(".workspace-option.add").trigger("click");
    expect(wrapper.find('[data-testid="workspace-create-modal"]').exists()).toBe(true);
  });
});

function mountSidebar() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:pathMatch(.*)*", component: { template: "<div />" } }]
  });

  return mount(LocalSettingsSidebar, {
    props: {
      activeKey: "settings_theme",
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
