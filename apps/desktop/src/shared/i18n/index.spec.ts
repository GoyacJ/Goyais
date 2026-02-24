import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("i18n locale persistence", () => {
  beforeEach(() => {
    vi.resetModules();
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it("优先读取持久化 locale", async () => {
    window.localStorage.setItem("goyais.locale", "en-US");
    const mod = await import("@/shared/i18n");
    expect(mod.i18nState.locale).toBe("en-US");
  });

  it("setLocale 会持久化到 localStorage", async () => {
    const mod = await import("@/shared/i18n");
    mod.setLocale("zh-CN");
    expect(window.localStorage.getItem("goyais.locale")).toBe("zh-CN");
  });
});
