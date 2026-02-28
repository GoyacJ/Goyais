import { beforeEach, describe, expect, it, vi } from "vitest";

const runtimeMocks = vi.hoisted(() => ({
  isRuntimeCapabilitySupported: vi.fn(() => true)
}));

const tauriCoreMocks = vi.hoisted(() => ({
  invoke: vi.fn(async () => "zoom")
}));

const tauriWindowMocks = vi.hoisted(() => {
  const appWindow = {
    close: vi.fn(async () => {}),
    minimize: vi.fn(async () => {}),
    maximize: vi.fn(async () => {}),
    unmaximize: vi.fn(async () => {}),
    isMaximized: vi.fn(async () => false),
    toggleMaximize: vi.fn(async () => {}),
    startDragging: vi.fn(async () => {})
  };

  return {
    appWindow,
    getCurrentWindow: vi.fn(() => appWindow)
  };
});

vi.mock("@/shared/runtime", () => ({
  isRuntimeCapabilitySupported: runtimeMocks.isRuntimeCapabilitySupported
}));

vi.mock("@tauri-apps/api/window", () => ({
  getCurrentWindow: tauriWindowMocks.getCurrentWindow
}));

vi.mock("@tauri-apps/api/core", () => ({
  invoke: tauriCoreMocks.invoke
}));

describe("windowControls handleTitlebarMouseDown", () => {
  beforeEach(() => {
    vi.resetModules();
    runtimeMocks.isRuntimeCapabilitySupported.mockReset();
    runtimeMocks.isRuntimeCapabilitySupported.mockReturnValue(true);
    tauriCoreMocks.invoke.mockReset();
    tauriCoreMocks.invoke.mockResolvedValue("zoom");
    tauriWindowMocks.getCurrentWindow.mockClear();
    tauriWindowMocks.appWindow.close.mockClear();
    tauriWindowMocks.appWindow.minimize.mockClear();
    tauriWindowMocks.appWindow.maximize.mockClear();
    tauriWindowMocks.appWindow.unmaximize.mockClear();
    tauriWindowMocks.appWindow.isMaximized.mockClear();
    tauriWindowMocks.appWindow.toggleMaximize.mockClear();
    tauriWindowMocks.appWindow.startDragging.mockClear();
  });

  it("double click on non-interactive region zooms when system preference is zoom", async () => {
    const { handleTitlebarMouseDown } = await import("@/shared/services/windowControls");
    const target = document.createElement("div");
    const event = createMouseDownEvent(target, { button: 0, detail: 2 });

    await handleTitlebarMouseDown(event);

    expect(tauriCoreMocks.invoke).toHaveBeenCalledWith("get_macos_titlebar_double_click_action");
    expect(tauriWindowMocks.appWindow.toggleMaximize).toHaveBeenCalledTimes(1);
    expect(tauriWindowMocks.appWindow.minimize).not.toHaveBeenCalled();
    expect(tauriWindowMocks.appWindow.startDragging).not.toHaveBeenCalled();
  });

  it("double click on non-interactive region minimizes when system preference is minimize", async () => {
    tauriCoreMocks.invoke.mockResolvedValueOnce("minimize");

    const { handleTitlebarMouseDown } = await import("@/shared/services/windowControls");
    const target = document.createElement("div");
    const event = createMouseDownEvent(target, { button: 0, detail: 2 });

    await handleTitlebarMouseDown(event);

    expect(tauriCoreMocks.invoke).toHaveBeenCalledWith("get_macos_titlebar_double_click_action");
    expect(tauriWindowMocks.appWindow.minimize).toHaveBeenCalledTimes(1);
    expect(tauriWindowMocks.appWindow.toggleMaximize).not.toHaveBeenCalled();
    expect(tauriWindowMocks.appWindow.startDragging).not.toHaveBeenCalled();
  });

  it("single click on non-interactive region starts dragging", async () => {
    const { handleTitlebarMouseDown } = await import("@/shared/services/windowControls");
    const target = document.createElement("div");
    const event = createMouseDownEvent(target, { button: 0, detail: 1 });

    await handleTitlebarMouseDown(event);

    expect(tauriWindowMocks.appWindow.startDragging).toHaveBeenCalledTimes(1);
    expect(tauriWindowMocks.appWindow.toggleMaximize).not.toHaveBeenCalled();
  });

  it("does nothing when target is interactive element", async () => {
    const { handleTitlebarMouseDown } = await import("@/shared/services/windowControls");
    const button = document.createElement("button");
    const event = createMouseDownEvent(button, { button: 0, detail: 2 });

    await handleTitlebarMouseDown(event);

    expect(tauriWindowMocks.appWindow.startDragging).not.toHaveBeenCalled();
    expect(tauriWindowMocks.appWindow.toggleMaximize).not.toHaveBeenCalled();
  });

  it("does nothing when window controls capability is unavailable", async () => {
    runtimeMocks.isRuntimeCapabilitySupported.mockReturnValue(false);

    const { handleTitlebarMouseDown } = await import("@/shared/services/windowControls");
    const target = document.createElement("div");
    const event = createMouseDownEvent(target, { button: 0, detail: 2 });

    await handleTitlebarMouseDown(event);

    expect(tauriCoreMocks.invoke).not.toHaveBeenCalled();
    expect(tauriWindowMocks.appWindow.minimize).not.toHaveBeenCalled();
    expect(tauriWindowMocks.appWindow.toggleMaximize).not.toHaveBeenCalled();
    expect(tauriWindowMocks.appWindow.startDragging).not.toHaveBeenCalled();
  });
});

function createMouseDownEvent(target: HTMLElement, init: MouseEventInit): MouseEvent {
  const event = new MouseEvent("mousedown", {
    bubbles: true,
    ...init
  });
  Object.defineProperty(event, "target", {
    configurable: true,
    value: target
  });
  return event;
}
