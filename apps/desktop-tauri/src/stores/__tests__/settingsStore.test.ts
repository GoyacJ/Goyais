import { beforeEach, describe, expect, it } from "vitest";

import { useSettingsStore } from "@/stores/settingsStore";

describe("settingsStore locale", () => {
  beforeEach(() => {
    localStorage.clear();
    useSettingsStore.setState({
      runtimeUrl: "http://127.0.0.1:8040",
      locale: "zh-CN"
    });
  });

  it("setLocale persists value", async () => {
    await useSettingsStore.getState().setLocale("en-US");

    expect(useSettingsStore.getState().locale).toBe("en-US");
    expect(localStorage.getItem("goyais.locale")).toBe("en-US");
  });

  it("hydrateLocale loads stored value", async () => {
    localStorage.setItem("goyais.locale", "en-US");

    await useSettingsStore.getState().hydrateLocale();

    expect(useSettingsStore.getState().locale).toBe("en-US");
  });
});
