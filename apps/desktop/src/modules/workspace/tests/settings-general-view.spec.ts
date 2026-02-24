import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it } from "vitest";

import SettingsGeneralView from "@/modules/workspace/views/SettingsGeneralView.vue";
import { setLocale } from "@/shared/i18n";
import { resetAuthStore } from "@/shared/stores/authStore";
import { resetWorkspaceStore } from "@/shared/stores/workspaceStore";

describe("settings general view", () => {
  beforeEach(async () => {
    window.localStorage.clear();
    resetWorkspaceStore();
    resetAuthStore();
    setLocale("zh-CN");
    const mod = await import("@/modules/workspace/store/generalSettingsStore");
    mod.resetGeneralSettingsStoreForTest();
  });

  it("renders compact section+row layout with six groups", async () => {
    const wrapper = mountView();
    await flushPromises();

    expect(wrapper.findAll(".settings-section")).toHaveLength(6);
    expect(wrapper.findAll(".row").length).toBeGreaterThanOrEqual(10);
    expect(wrapper.find(".footer-actions").exists()).toBe(true);
  });

  it("shows unsupported hints when platform capability is unavailable", async () => {
    const wrapper = mountView();
    await flushPromises();

    expect(wrapper.text()).toContain("当前平台暂不支持该系统能力。");
  });

  it("renders current version and check version action in update policy section", async () => {
    const wrapper = mountView();
    await flushPromises();

    expect(wrapper.text()).toContain("当前版本");
    expect(wrapper.text()).toContain("检查版本");
    expect(wrapper.findAll("button").some((item) => item.text() === "检查版本")).toBe(true);
  });
});

function mountView() {
  return mount(SettingsGeneralView, {
    global: {
      stubs: {
        SettingsShell: {
          template: "<div class='settings-shell-stub'><slot /></div>"
        }
      }
    }
  });
}
