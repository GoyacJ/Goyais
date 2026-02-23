import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import SettingsThemeView from "@/modules/workspace/views/SettingsThemeView.vue";
import { setLocale } from "@/shared/i18n";
import { initializeTheme, resetThemeSettings } from "@/shared/stores/themeStore";

const THEME_SETTINGS_STORAGE_KEY = "goyais.theme.settings.v1";

function mockMatchMedia(matches = false): void {
  vi.stubGlobal(
    "matchMedia",
    vi.fn().mockImplementation(() => ({
      matches,
      media: "(prefers-color-scheme: light)",
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn()
    }))
  );
}

function clearThemeAttributes(): void {
  document.documentElement.removeAttribute("data-theme");
  document.documentElement.removeAttribute("data-theme-mode");
  document.documentElement.removeAttribute("data-font-style");
  document.documentElement.removeAttribute("data-font-scale");
  document.documentElement.removeAttribute("data-theme-preset");
}

describe("settings theme view", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    window.localStorage.clear();
    clearThemeAttributes();
    mockMatchMedia(false);
    setLocale("zh-CN");
    initializeTheme();
    resetThemeSettings();
  });

  it("renders config panel without live preview", () => {
    const wrapper = mountView();

    expect(wrapper.find(".theme-layout").exists()).toBe(true);
    expect(wrapper.find(".config-panel").exists()).toBe(true);
    expect(wrapper.find(".preview-panel").exists()).toBe(false);
    expect(wrapper.text()).not.toContain("实时预览");
    expect(wrapper.findAll(".config-group")).toHaveLength(4);
  });

  it("shows four configurable sections with zh-CN labels", () => {
    const wrapper = mountView();

    expect(wrapper.text()).toContain("主题");
    expect(wrapper.text()).toContain("字体样式");
    expect(wrapper.text()).toContain("字体大小");
    expect(wrapper.text()).toContain("预设主题");
    expect(wrapper.text()).toContain("恢复默认");
    expect(wrapper.text()).not.toContain("深色模式");
    expect(wrapper.text()).not.toContain("主题、字体与排版设置会立即作用于整个应用。");
    expect(wrapper.text()).not.toContain("当前生效主题");
    expect(wrapper.text()).not.toContain("当前预设");
    expect(wrapper.text()).not.toContain("切换后即时生效并自动持久化。");

    const modeOptions = wrapper.findAll('[data-testid="theme-mode-select"] option').map((item) => item.text());
    expect(modeOptions).toEqual(["跟随系统", "深色", "浅色"]);
  });

  it("updates theme attributes and localStorage when controls change", async () => {
    const wrapper = mountView();

    await wrapper.get('[data-testid="theme-mode-select"] select').setValue("dark");
    await wrapper.get('[data-testid="font-style-select"] select').setValue("coding");
    await wrapper.get('[data-testid="font-scale-select"] select').setValue("sm");

    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    expect(document.documentElement.getAttribute("data-theme-mode")).toBe("dark");
    expect(document.documentElement.getAttribute("data-font-style")).toBe("coding");
    expect(document.documentElement.getAttribute("data-font-scale")).toBe("sm");
    expect(document.documentElement.getAttribute("data-theme-preset")).toBe("obsidian_pulse");

    const persisted = JSON.parse(window.localStorage.getItem(THEME_SETTINGS_STORAGE_KEY) ?? "{}");
    expect(persisted).toEqual({
      mode: "dark",
      fontStyle: "coding",
      fontScale: "sm",
      preset: "obsidian_pulse"
    });
  });

  it("applies preset and then resets to defaults", async () => {
    const wrapper = mountView();

    await wrapper.get('[data-testid="theme-preset-select"] select').setValue("paper_focus");

    expect(document.documentElement.getAttribute("data-theme")).toBe("light");
    expect(document.documentElement.getAttribute("data-font-style")).toBe("reading");
    expect(document.documentElement.getAttribute("data-font-scale")).toBe("lg");
    expect(document.documentElement.getAttribute("data-theme-preset")).toBe("paper_focus");

    await wrapper.get('[data-testid="theme-reset-button"]').trigger("click");

    expect(document.documentElement.getAttribute("data-theme-mode")).toBe("system");
    expect(document.documentElement.getAttribute("data-font-style")).toBe("neutral");
    expect(document.documentElement.getAttribute("data-font-scale")).toBe("md");
    expect(document.documentElement.getAttribute("data-theme-preset")).toBe("aurora_forge");

    expect((wrapper.get('[data-testid="theme-mode-select"] select').element as HTMLSelectElement).value).toBe("system");
  });
});

function mountView() {
  return mount(SettingsThemeView, {
    global: {
      stubs: {
        SettingsShell: {
          template: '<div class="settings-shell-stub"><slot /></div>'
        }
      }
    }
  });
}
