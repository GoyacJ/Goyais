type TauriWindow = {
  close: () => Promise<void>;
  minimize: () => Promise<void>;
  maximize: () => Promise<void>;
  unmaximize: () => Promise<void>;
  isMaximized: () => Promise<boolean>;
  toggleMaximize?: () => Promise<void>;
  startDragging: () => Promise<void>;
};

const INTERACTIVE_SELECTOR = "button,a,input,select,textarea,[role='button'],[data-no-drag='true']";

let cachedWindow: Promise<TauriWindow | null> | null = null;

export async function closeCurrentWindow(): Promise<void> {
  await withWindowAction((appWindow) => appWindow.close(), () => {
    if (typeof window !== "undefined" && typeof window.close === "function") {
      window.close();
    }
  });
}

export async function minimizeCurrentWindow(): Promise<void> {
  await withWindowAction((appWindow) => appWindow.minimize());
}

export async function toggleMaximizeCurrentWindow(): Promise<void> {
  await withWindowAction(async (appWindow) => {
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
  await withWindowAction((appWindow) => appWindow.startDragging());
}

export async function handleDragMouseDown(event: MouseEvent): Promise<void> {
  if (event.button !== 0) {
    return;
  }

  if (isInteractiveTarget(event.target)) {
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

async function withWindowAction(action: (appWindow: TauriWindow) => Promise<void>, fallback?: () => void): Promise<void> {
  const appWindow = await getCurrentTauriWindow();
  if (!appWindow) {
    fallback?.();
    return;
  }

  try {
    await action(appWindow);
  } catch {
    fallback?.();
  }
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
