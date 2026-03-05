import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it } from "vitest";

import { setLocale } from "@/shared/i18n";
import ConfigTopStatusChips from "@/shared/ui/ConfigTopStatusChips.vue";

describe("config top status chips", () => {
  beforeEach(() => {
    setLocale("en-US");
  });

  it("renders runtime conversation status", () => {
    const wrapper = mount(ConfigTopStatusChips, {
      props: {
        runtimeMode: true,
        conversationStatus: "running",
        scopeMode: "local"
      }
    });

    expect(wrapper.text()).toContain("session: running");
    expect(wrapper.text()).not.toContain("scope:");
  });

  it("renders local non-runtime scope and mode with zh labels", () => {
    setLocale("zh-CN");

    const wrapper = mount(ConfigTopStatusChips, {
      props: {
        runtimeMode: false,
        scopeMode: "local"
      }
    });

    expect(wrapper.text()).toContain("scope: 本地工作区");
    expect(wrapper.text()).toContain("本地");
  });

  it("renders remote non-runtime scope and mode", () => {
    const wrapper = mount(ConfigTopStatusChips, {
      props: {
        runtimeMode: false,
        scopeMode: "remote"
      }
    });

    expect(wrapper.text()).toContain("scope: current_workspace");
    expect(wrapper.text()).toContain("Remote");
  });
});
