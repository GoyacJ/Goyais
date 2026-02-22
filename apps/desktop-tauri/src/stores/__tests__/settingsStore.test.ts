import { beforeEach, describe, expect, it } from "vitest";

import { useSettingsStore } from "@/stores/settingsStore";
import { createDefaultLocalProcessConfig } from "@/types/localProcessConfig";

describe("settingsStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useSettingsStore.setState({
      locale: "zh-CN",
      theme: "dark",
      defaultModelConfigId: undefined,
      localProcessConfig: createDefaultLocalProcessConfig()
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

  it("updates local process config and marks pending apply", () => {
    useSettingsStore.getState().setLocalProcessConfig((current) => ({
      ...current,
      hub: {
        ...current.hub,
        port: "9001"
      },
      pendingApply: {
        ...current.pendingApply,
        hub: true
      }
    }));

    expect(useSettingsStore.getState().localProcessConfig.hub.port).toBe("9001");
    expect(useSettingsStore.getState().localProcessConfig.pendingApply.hub).toBe(true);
  });

  it("removes legacy runtime/local auth keys during hydrate", async () => {
    localStorage.setItem("goyais.runtimeUrl", "http://127.0.0.1:8040");
    localStorage.setItem("goyais.localAutoPassword", "legacy");
    localStorage.setItem("goyais.localHubUrl", "127.0.0.1:9000");

    await useSettingsStore.getState().hydrate();

    expect(localStorage.getItem("goyais.runtimeUrl")).toBeNull();
    expect(localStorage.getItem("goyais.localAutoPassword")).toBeNull();
    expect(useSettingsStore.getState().localProcessConfig.connections.localHubUrl).toBe("http://127.0.0.1:9000");
  });
});
