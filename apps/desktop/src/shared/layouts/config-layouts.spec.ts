import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";

import type { MenuEntry } from "@/shared/navigation/pageMenus";
import LocalSettingsLayout from "@/shared/layouts/LocalSettingsLayout.vue";
import RemoteConfigLayout from "@/shared/layouts/RemoteConfigLayout.vue";

const menuEntries: MenuEntry[] = [];

const sharedStubs = {
  LocalSettingsSidebar: { template: '<aside class="stub-local-sidebar">local</aside>' },
  RemoteConfigSidebar: { template: '<aside class="stub-remote-sidebar">remote</aside>' },
  Topbar: { template: '<header class="stub-topbar"><slot name="left" /><slot name="right" /></header>' },
  HubStatusBar: { template: '<footer class="stub-status"></footer>' },
  StatusBadge: { template: '<span class="stub-badge"></span>' }
};

describe("config layouts", () => {
  it("local settings layout keeps a stretch container around the sidebar", () => {
    const wrapper = mount(LocalSettingsLayout, {
      props: {
        title: "local",
        subtitle: "settings",
        activeKey: "settings_theme",
        menuEntries
      },
      slots: {
        default: '<div class="local-page">content</div>'
      },
      global: {
        stubs: sharedStubs
      }
    });

    expect(wrapper.find(".sidebar-slot-fill").exists()).toBe(true);
  });

  it("remote config layout keeps a stretch container around the sidebar", () => {
    const wrapper = mount(RemoteConfigLayout, {
      props: {
        title: "remote",
        subtitle: "account",
        scopeHint: "workspace",
        activeKey: "remote_account",
        menuEntries
      },
      slots: {
        default: '<div class="remote-page">content</div>'
      },
      global: {
        stubs: sharedStubs
      }
    });

    expect(wrapper.find(".sidebar-slot-fill").exists()).toBe(true);
  });

  it("local settings layout keeps fixed viewport height so content panel can scroll", () => {
    const source = readFileSync(join(process.cwd(), "src/shared/layouts/LocalSettingsLayout.vue"), "utf8");
    const layoutBlock = source.match(/\.layout\s*\{[\s\S]*?\}/)?.[0] ?? "";
    const declarations = layoutBlock.split("\n").map((line) => line.trim());

    expect(declarations).toContain("height: 100dvh;");
  });

  it("remote config layout keeps fixed viewport height so content panel can scroll", () => {
    const source = readFileSync(join(process.cwd(), "src/shared/layouts/RemoteConfigLayout.vue"), "utf8");
    const layoutBlock = source.match(/\.layout\s*\{[\s\S]*?\}/)?.[0] ?? "";
    const declarations = layoutBlock.split("\n").map((line) => line.trim());

    expect(declarations).toContain("height: 100dvh;");
  });
});
