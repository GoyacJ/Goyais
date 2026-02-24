import { beforeEach, describe, expect, it, vi } from "vitest";

const THEME_SETTINGS_STORAGE_KEY = "goyais.theme.settings.v1";

type MatchMediaControl = {
  setMatches(next: boolean): void;
  emitChange(): void;
};

function mockMatchMedia(matches = false): MatchMediaControl {
  let changeHandler: ((event: MediaQueryListEvent) => void) | null = null;
  const mediaQuery = {
    matches,
    media: "(prefers-color-scheme: light)",
    onchange: null,
    addEventListener: vi.fn((event: string, handler: EventListenerOrEventListenerObject) => {
      if (event !== "change") {
        return;
      }
      if (typeof handler === "function") {
        changeHandler = handler as (event: MediaQueryListEvent) => void;
      }
    }),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn()
  } as unknown as MediaQueryList;

  vi.stubGlobal(
    "matchMedia",
    vi.fn().mockImplementation(() => mediaQuery)
  );

  return {
    setMatches(next: boolean) {
      Object.assign(mediaQuery, { matches: next });
    },
    emitChange() {
      if (changeHandler) {
        changeHandler({ matches: (mediaQuery as { matches: boolean }).matches } as MediaQueryListEvent);
      }
    }
  };
}

function clearThemeAttributes(): void {
  document.documentElement.removeAttribute("data-theme");
  document.documentElement.removeAttribute("data-theme-mode");
  document.documentElement.removeAttribute("data-font-style");
  document.documentElement.removeAttribute("data-font-scale");
  document.documentElement.removeAttribute("data-theme-preset");
}

describe("theme store", () => {
  beforeEach(() => {
    vi.resetModules();
    window.localStorage.clear();
    clearThemeAttributes();
  });

  it("initializeTheme 会读取新配置并应用全部主题属性", async () => {
    window.localStorage.setItem(
      THEME_SETTINGS_STORAGE_KEY,
      JSON.stringify({
        mode: "light",
        fontStyle: "coding",
        fontScale: "lg",
        preset: "paper_focus"
      })
    );
    mockMatchMedia(false);

    const mod = await import("@/shared/stores/themeStore");
    mod.initializeTheme();

    expect(document.documentElement.getAttribute("data-theme")).toBe("light");
    expect(document.documentElement.getAttribute("data-theme-mode")).toBe("light");
    expect(document.documentElement.getAttribute("data-font-style")).toBe("coding");
    expect(document.documentElement.getAttribute("data-font-scale")).toBe("lg");
    expect(document.documentElement.getAttribute("data-theme-preset")).toBe("paper_focus");
  });

  it("system 模式会跟随系统亮暗变化", async () => {
    const media = mockMatchMedia(false);
    const mod = await import("@/shared/stores/themeStore");
    mod.initializeTheme();

    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    media.setMatches(true);
    media.emitChange();
    expect(document.documentElement.getAttribute("data-theme")).toBe("light");
  });

  it("setThemePreset 会批量更新 mode/style/scale 并持久化", async () => {
    mockMatchMedia(false);
    const mod = await import("@/shared/stores/themeStore");
    mod.initializeTheme();

    mod.setThemePreset("paper_focus");

    expect(document.documentElement.getAttribute("data-theme")).toBe("light");
    expect(document.documentElement.getAttribute("data-theme-mode")).toBe("light");
    expect(document.documentElement.getAttribute("data-font-style")).toBe("reading");
    expect(document.documentElement.getAttribute("data-font-scale")).toBe("lg");
    expect(document.documentElement.getAttribute("data-theme-preset")).toBe("paper_focus");

    const persisted = JSON.parse(window.localStorage.getItem(THEME_SETTINGS_STORAGE_KEY) ?? "{}");
    expect(persisted).toEqual({
      mode: "light",
      fontStyle: "reading",
      fontScale: "lg",
      preset: "paper_focus"
    });
  });

  it("setThemePreference 兼容别名会继续生效", async () => {
    mockMatchMedia(false);
    const mod = await import("@/shared/stores/themeStore");
    mod.initializeTheme();

    mod.setThemePreference("dark");

    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    expect(document.documentElement.getAttribute("data-theme-mode")).toBe("dark");
  });

  it("resetThemeSettings 会恢复默认配置", async () => {
    mockMatchMedia(false);
    const mod = await import("@/shared/stores/themeStore");
    mod.initializeTheme();

    mod.setThemePreset("paper_focus");
    mod.resetThemeSettings();

    expect(document.documentElement.getAttribute("data-theme-mode")).toBe("system");
    expect(document.documentElement.getAttribute("data-font-style")).toBe("neutral");
    expect(document.documentElement.getAttribute("data-font-scale")).toBe("md");
    expect(document.documentElement.getAttribute("data-theme-preset")).toBe("aurora_forge");
  });

  it("不会读取旧键 goyais.theme.preference", async () => {
    window.localStorage.setItem("goyais.theme.preference", "light");
    mockMatchMedia(false);
    const mod = await import("@/shared/stores/themeStore");
    mod.initializeTheme();

    const theme = mod.useTheme();
    expect(theme.mode.value).toBe("system");
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    expect(window.localStorage.getItem(THEME_SETTINGS_STORAGE_KEY)).toBeNull();
  });
});
