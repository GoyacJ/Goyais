import { beforeEach, describe, expect, it, vi } from "vitest";

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

describe("theme store", () => {
  beforeEach(() => {
    vi.resetModules();
    window.localStorage.clear();
    document.documentElement.removeAttribute("data-theme");
  });

  it("initializeTheme 会读取本地偏好并应用", async () => {
    window.localStorage.setItem("goyais.theme.preference", "light");
    mockMatchMedia(false);

    const mod = await import("@/shared/stores/themeStore");
    mod.initializeTheme();

    expect(document.documentElement.getAttribute("data-theme")).toBe("light");
  });

  it("setThemePreference 会更新 data-theme 与持久化", async () => {
    mockMatchMedia(false);
    const mod = await import("@/shared/stores/themeStore");
    mod.initializeTheme();

    mod.setThemePreference("dark");

    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    expect(window.localStorage.getItem("goyais.theme.preference")).toBe("dark");
  });
});
