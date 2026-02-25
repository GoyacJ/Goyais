import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";

import MainShell from "@/shared/shells/MainShell.vue";

describe("main shell", () => {
  it("renders main shell slots with shared layout containers", () => {
    const wrapper = mount(MainShell, {
      slots: {
        sidebar: '<aside class="slot-sidebar">sidebar</aside>',
        header: '<header class="slot-header">header</header>',
        main: '<main class="slot-main">main</main>',
        footer: '<footer class="slot-footer">footer</footer>'
      }
    });

    expect(wrapper.find(".screen").exists()).toBe(true);
    expect(wrapper.find(".content").exists()).toBe(true);
    expect(wrapper.find(".slot-sidebar").exists()).toBe(true);
    expect(wrapper.find(".slot-header").exists()).toBe(true);
    expect(wrapper.find(".slot-main").exists()).toBe(true);
    expect(wrapper.find(".slot-footer").exists()).toBe(true);
  });

  it("wraps sidebar slot with a stretch container to preserve full-height desktop sidebar", () => {
    const wrapper = mount(MainShell, {
      slots: {
        sidebar: '<aside class="slot-sidebar">sidebar</aside>'
      }
    });

    expect(wrapper.find(".sidebar-slot-fill").exists()).toBe(true);
  });
});
