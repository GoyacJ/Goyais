import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it } from "vitest";

import SettingsI18nView from "@/modules/workspace/views/SettingsI18nView.vue";
import { i18nState, setLocale } from "@/shared/i18n";
import { resetAuthStore } from "@/shared/stores/authStore";
import { resetWorkspaceStore } from "@/shared/stores/workspaceStore";

describe("settings i18n view", () => {
  beforeEach(() => {
    window.localStorage.clear();
    resetWorkspaceStore();
    resetAuthStore();
    setLocale("zh-CN");
  });

  it("renders a single main card without preview block", () => {
    const wrapper = mountView();

    expect(wrapper.findAll(".settings-panel")).toHaveLength(1);
    expect(wrapper.text()).not.toContain("文案预览");
    expect(wrapper.text()).not.toContain("输入消息，支持 @resource/command");
  });

  it("shows locale options in native-name-plus-code format", () => {
    const wrapper = mountView();
    const options = wrapper.findAll("option").map((node) => node.text());

    expect(options).toEqual(["简体中文（zh-CN）", "English (US)（en-US）"]);
  });

  it("updates locale and persists when switching", async () => {
    const wrapper = mountView();
    await wrapper.get("select").setValue("en-US");

    expect(i18nState.locale).toBe("en-US");
    expect(window.localStorage.getItem("goyais.locale")).toBe("en-US");
  });

  it("does not render current locale summary text", () => {
    setLocale("en-US");
    const wrapper = mountView();

    expect(wrapper.find(".inline-meta").exists()).toBe(false);
    expect(wrapper.text()).not.toContain("Current Locale:");
    expect(wrapper.text()).not.toContain("当前语言:");
  });
});

function mountView() {
  return mount(SettingsI18nView, {
    global: {
      stubs: {
        SettingsShell: {
          template: "<div class=\"settings-shell-stub\"><slot /></div>"
        }
      }
    }
  });
}
