import { beforeEach, describe, expect, it } from "vitest";

import { useSettingsStore } from "@/stores/settingsStore";

describe("settingsStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useSettingsStore.setState({
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

  it("hydrates locale from storage", async () => {
    localStorage.setItem("goyais.locale", "en-US");

    await useSettingsStore.getState().hydrate();

    expect(useSettingsStore.getState().locale).toBe("en-US");
  });

  it("updates theme", () => {
    useSettingsStore.getState().setTheme("light");
    expect(useSettingsStore.getState().theme).toBe("light");
  });

  it("updates default model config", () => {
    useSettingsStore.getState().setDefaultModelConfigId("mc-1");
    expect(useSettingsStore.getState().defaultModelConfigId).toBe("mc-1");
  });

  it("removes legacy runtime/local auth keys during hydrate", async () => {
    localStorage.setItem("goyais.runtimeUrl", "http://127.0.0.1:8040");
    localStorage.setItem("goyais.localAutoPassword", "legacy");

    await useSettingsStore.getState().hydrate();

    expect(localStorage.getItem("goyais.runtimeUrl")).toBeNull();
    expect(localStorage.getItem("goyais.localAutoPassword")).toBeNull();
  });
});
