import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  load: vi.fn(),
  save: vi.fn(),
  detect: vi.fn(),
  apply: vi.fn()
}));

vi.mock("@/modules/workspace/services/generalSettingsPersistence", () => ({
  loadGeneralSettings: mocks.load,
  saveGeneralSettings: mocks.save
}));

vi.mock("@/modules/workspace/services/generalSettingsPlatform", () => ({
  detectGeneralSettingsCapability: mocks.detect,
  applyGeneralSettingsField: mocks.apply,
  resolveUpdatePolicyNextCheck: vi.fn(),
  resolveDiagnosticsRetentionDeadline: vi.fn()
}));

describe("general settings store", () => {
  beforeEach(() => {
    vi.resetModules();
    mocks.load.mockReset();
    mocks.save.mockReset();
    mocks.detect.mockReset();
    mocks.apply.mockReset();
    mocks.detect.mockResolvedValue(createCapability(false, false));
    mocks.apply.mockResolvedValue(undefined);
  });

  it("uses defaults when no persisted settings exist", async () => {
    mocks.load.mockResolvedValue(null);
    const mod = await import("@/modules/workspace/store/generalSettingsStore");

    await mod.initializeGeneralSettings();

    const settings = mod.useGeneralSettings();
    expect(settings.state.value.defaultProjectDirectory).toBe("~/.goyais");
    expect(settings.state.value.telemetryLevel).toBe("minimized");
    expect(settings.state.value.updatePolicy.checkFrequency).toBe("daily");
  });

  it("updates a field and persists immediately", async () => {
    mocks.load.mockResolvedValue(null);
    mocks.detect.mockResolvedValue(createCapability(true, true));

    const mod = await import("@/modules/workspace/store/generalSettingsStore");
    await mod.initializeGeneralSettings();

    const settings = mod.useGeneralSettings();
    await settings.updateField("telemetryLevel", "off");

    expect(settings.state.value.telemetryLevel).toBe("off");
    expect(mocks.save).toHaveBeenCalled();
    expect(mocks.apply).toHaveBeenCalledWith(
      "telemetryLevel",
      expect.objectContaining({ telemetryLevel: "off" }),
      expect.any(Object)
    );
  });

  it("falls back to defaults and records error when load fails", async () => {
    mocks.load.mockRejectedValue(new Error("load failed"));

    const mod = await import("@/modules/workspace/store/generalSettingsStore");
    await mod.initializeGeneralSettings();

    const settings = mod.useGeneralSettings();
    expect(settings.state.value.defaultProjectDirectory).toBe("~/.goyais");
    expect(settings.error.value).toContain("load failed");
  });

  it("resetAll restores defaults and persists", async () => {
    mocks.load.mockResolvedValue(null);

    const mod = await import("@/modules/workspace/store/generalSettingsStore");
    await mod.initializeGeneralSettings();

    const settings = mod.useGeneralSettings();
    await settings.updateField("defaultProjectDirectory", "/tmp/demo");
    await settings.resetAll();

    expect(settings.state.value.defaultProjectDirectory).toBe("~/.goyais");
    expect(settings.state.value.diagnostics.logRetentionDays).toBe(14);
    expect(mocks.save).toHaveBeenCalledTimes(2);
  });
});

function createCapability(launch: boolean, notifications: boolean) {
  return {
    launchOnStartup: {
      supported: launch,
      reasonKey: launch ? "" : "settings.general.unsupported.platform"
    },
    notifications: {
      supported: notifications,
      reasonKey: notifications ? "" : "settings.general.unsupported.notifications"
    }
  };
}
