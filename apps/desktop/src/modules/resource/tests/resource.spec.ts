import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it } from "vitest";

import { resetProjectStore } from "@/modules/project/store";
import { resetResourceStore } from "@/modules/resource/store";
import WorkspaceMcpView from "@/modules/resource/views/WorkspaceMcpView.vue";
import WorkspaceModelView from "@/modules/resource/views/WorkspaceModelView.vue";
import WorkspaceProjectConfigView from "@/modules/resource/views/WorkspaceProjectConfigView.vue";
import WorkspaceRulesView from "@/modules/resource/views/WorkspaceRulesView.vue";
import { resetAuthStore } from "@/shared/stores/authStore";
import { resetWorkspaceStore, setCurrentWorkspace, setWorkspaces } from "@/shared/stores/workspaceStore";

describe("resource module views", () => {
  beforeEach(() => {
    resetWorkspaceStore();
    resetAuthStore();
    resetProjectStore();
    resetResourceStore();

    const now = new Date().toISOString();
    setWorkspaces([
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
    ]);
    setCurrentWorkspace("ws_local");
  });

  it("renders model config table and modal action", async () => {
    const wrapper = mountView(WorkspaceModelView);
    await flushPromises();

    expect(wrapper.text()).toContain("模型列表");
    const addButton = wrapper.findAll("button").find((item) => item.text() === "新增模型");
    expect(addButton).toBeTruthy();
    await addButton?.trigger("click");
    expect(wrapper.text()).toContain("新增模型配置");
  });

  it("renders rules page with markdown editor modal", async () => {
    const wrapper = mountView(WorkspaceRulesView);
    await flushPromises();

    expect(wrapper.text()).toContain("规则列表");
    const addButton = wrapper.findAll("button").find((item) => item.text() === "新增规则");
    expect(addButton).toBeTruthy();
    await addButton?.trigger("click");
    expect(wrapper.text()).toContain("新增规则");
  });

  it("renders mcp cards page and export action", async () => {
    const wrapper = mountView(WorkspaceMcpView);
    await flushPromises();

    expect(wrapper.text()).toContain("查看聚合 JSON");
  });

  it("renders project config table", async () => {
    const wrapper = mountView(WorkspaceProjectConfigView);
    await flushPromises();

    expect(wrapper.text()).toContain("项目导入与绑定");
    expect(wrapper.text()).toContain("目录导入");
  });
});

function mountView(component: unknown) {
  return mount(component as never, {
    global: {
      stubs: {
        WorkspaceSharedShell: {
          template: "<div class='workspace-shared-shell-stub'><slot /></div>"
        }
      }
    }
  });
}
