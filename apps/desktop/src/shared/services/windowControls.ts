import { isRuntimeCapabilitySupported } from "@/shared/runtime";

type TauriWindow = {
  close: () => Promise<void>;
  minimize: () => Promise<void>;
  maximize: () => Promise<void>;
  unmaximize: () => Promise<void>;
  isMaximized: () => Promise<boolean>;
  toggleMaximize?: () => Promise<void>;
  startDragging: () => Promise<void>;
};

type TauriCoreModule = {
  invoke: <T>(command: string) => Promise<T>;
};

type TitlebarDoubleClickAction = "minimize" | "zoom";

const INTERACTIVE_SELECTOR = "button,a,input,select,textarea,[role='button'],[data-no-drag='true']";
const MACOS_TITLEBAR_DOUBLE_CLICK_ACTION_COMMAND = "get_macos_titlebar_double_click_action";
const DEFAULT_TITLEBAR_DOUBLE_CLICK_ACTION: TitlebarDoubleClickAction = "zoom";

let cachedWindow: Promise<TauriWindow | null> | null = null;
let cachedTitlebarDoubleClickAction: Promise<TitlebarDoubleClickAction> | null = null;

export async function closeCurrentWindow(): Promise<void> {
  if (!isRuntimeCapabilitySupported("supportsWindowControls")) {
    return;
  }

  await withWindowAction("close", (appWindow) => appWindow.close(), () => {
    if (typeof window !== "undefined" && typeof window.close === "function") {
      window.close();
    }
  });
}

export async function minimizeCurrentWindow(): Promise<void> {
  if (!isRuntimeCapabilitySupported("supportsWindowControls")) {
    return;
  }
  await withWindowAction("minimize", (appWindow) => appWindow.minimize());
}

export async function toggleMaximizeCurrentWindow(): Promise<void> {
  if (!isRuntimeCapabilitySupported("supportsWindowControls")) {
    return;
  }

  await withWindowAction("toggleMaximize", async (appWindow) => {
    if (typeof appWindow.toggleMaximize === "function") {
      await appWindow.toggleMaximize();
      return;
    }

    const maximized = await appWindow.isMaximized();
    if (maximized) {
      await appWindow.unmaximize();
      return;
    }

    await appWindow.maximize();
  }, toggleDocumentFullscreen);
}

export async function startCurrentWindowDragging(): Promise<void> {
  if (!isRuntimeCapabilitySupported("supportsWindowControls")) {
    return;
  }
  await withWindowAction("startDragging", (appWindow) => appWindow.startDragging());
}

export async function handleTitlebarMouseDown(event: MouseEvent): Promise<void> {
  if (event.button !== 0) {
    return;
  }

  if (isInteractiveTarget(event.target)) {
    return;
  }

  if (event.detail === 2) {
    const action = await resolveTitlebarDoubleClickAction();
    if (action === "minimize") {
      await minimizeCurrentWindow();
      return;
    }
    await toggleMaximizeCurrentWindow();
    return;
  }

  await startCurrentWindowDragging();
}

function isInteractiveTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }

  return target.closest(INTERACTIVE_SELECTOR) !== null;
}

async function getCurrentTauriWindow(): Promise<TauriWindow | null> {
  if (cachedWindow) {
    return cachedWindow;
  }

  cachedWindow = loadCurrentTauriWindow();
  return cachedWindow;
}

async function loadCurrentTauriWindow(): Promise<TauriWindow | null> {
  try {
    const windowModule = await import("@tauri-apps/api/window");
    return windowModule.getCurrentWindow() as unknown as TauriWindow;
  } catch {
    return null;
  }
}

async function resolveTitlebarDoubleClickAction(): Promise<TitlebarDoubleClickAction> {
  if (!isRuntimeCapabilitySupported("supportsWindowControls")) {
    return DEFAULT_TITLEBAR_DOUBLE_CLICK_ACTION;
  }

  if (cachedTitlebarDoubleClickAction) {
    return cachedTitlebarDoubleClickAction;
  }

  cachedTitlebarDoubleClickAction = loadTitlebarDoubleClickAction();
  return cachedTitlebarDoubleClickAction;
}

async function loadTitlebarDoubleClickAction(): Promise<TitlebarDoubleClickAction> {
  const coreModule = await loadTauriCoreModule();
  if (!coreModule) {
    reportWindowControlError(
      "resolveTitlebarDoubleClickAction",
      new Error("Tauri core module is unavailable")
    );
    return DEFAULT_TITLEBAR_DOUBLE_CLICK_ACTION;
  }

  try {
    const action = await coreModule.invoke<unknown>(MACOS_TITLEBAR_DOUBLE_CLICK_ACTION_COMMAND);
    return normalizeTitlebarDoubleClickAction(action);
  } catch (error) {
    reportWindowControlError("resolveTitlebarDoubleClickAction", error);
    return DEFAULT_TITLEBAR_DOUBLE_CLICK_ACTION;
  }
}

async function loadTauriCoreModule(): Promise<TauriCoreModule | null> {
  try {
    const coreModule = await import("@tauri-apps/api/core");
    return coreModule as TauriCoreModule;
  } catch {
    return null;
  }
}

function normalizeTitlebarDoubleClickAction(action: unknown): TitlebarDoubleClickAction {
  if (typeof action !== "string") {
    return DEFAULT_TITLEBAR_DOUBLE_CLICK_ACTION;
  }

  return action.trim().toLowerCase() === "minimize"
    ? "minimize"
    : DEFAULT_TITLEBAR_DOUBLE_CLICK_ACTION;
}

async function withWindowAction(actionName: string, action: (appWindow: TauriWindow) => Promise<void>, fallback?: () => void): Promise<void> {
  const appWindow = await getCurrentTauriWindow();
  if (!appWindow) {
    reportWindowControlError(actionName, new Error("Current Tauri window is unavailable"));
    fallback?.();
    return;
  }

  try {
    await action(appWindow);
  } catch (error) {
    reportWindowControlError(actionName, error);
    fallback?.();
  }
}

function reportWindowControlError(actionName: string, error: unknown): void {
  const message = `[windowControls] ${actionName} failed`;
  if (error instanceof Error) {
    console.error(message, error);
    return;
  }
  console.error(message, String(error));
}

function toggleDocumentFullscreen(): void {
  if (typeof document === "undefined") {
    return;
  }

  if (document.fullscreenElement) {
    void document.exitFullscreen();
    return;
  }

  void document.documentElement.requestFullscreen?.();
}
