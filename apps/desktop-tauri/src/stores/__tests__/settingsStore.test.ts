import { beforeEach, describe, expect, it } from "vitest";

import { RUNTIME_URL_STORAGE_KEY, useSettingsStore } from "@/stores/settingsStore";

describe("settingsStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useSettingsStore.setState({
      runtimeUrl: "http://127.0.0.1:8040",
      locale: "zh-CN",
      theme: "dark",
      defaultModelConfigId: undefined
    });
  });

  it("setLocale persists value", async () => {
    await useSettingsStore.getState().setLocale("en-US");

    expect(useSettingsStore.getState().locale).toBe("en-US");
    expect(localStorage.getItem("goyais.locale")).toBe("en-US");
  });

  it("setRuntimeUrl persists value", () => {
    useSettingsStore.getState().setRuntimeUrl("http://127.0.0.1:9000");

    expect(useSettingsStore.getState().runtimeUrl).toBe("http://127.0.0.1:9000");
    expect(localStorage.getItem(RUNTIME_URL_STORAGE_KEY)).toBe("http://127.0.0.1:9000");
  });

  it("rejects invalid runtime url", () => {
    expect(() => useSettingsStore.getState().setRuntimeUrl("ftp://127.0.0.1:9000")).toThrow();
    expect(useSettingsStore.getState().runtimeUrl).toBe("http://127.0.0.1:8040");
  });

  it("hydrates locale from storage", async () => {
    localStorage.setItem("goyais.locale", "en-US");

    await useSettingsStore.getState().hydrate();

    expect(useSettingsStore.getState().locale).toBe("en-US");
  });

  it("updates theme", () => {
    useSettingsStore.getState().setTheme("light");
    expect(useSettingsStore.getState().theme).toBe("light");
  });
});
