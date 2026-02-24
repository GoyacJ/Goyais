import { open as openDialog } from "@tauri-apps/plugin-dialog";

type DesktopDirectoryFile = File & {
  path?: string;
  webkitRelativePath?: string;
};

type DirectoryPickAttempt = {
  handled: boolean;
  value: string | null;
};

const FOCUS_FALLBACK_DELAY_MS = 1000;

export async function pickDirectoryPath(): Promise<string | null> {
  if (typeof document === "undefined" || typeof window === "undefined") {
    return null;
  }

  const tauriPicked = await tryPickDirectoryPathFromTauri();
  if (tauriPicked.handled) {
    if (tauriPicked.value) {
      return tauriPicked.value;
    }
    return pickDirectoryPathFromPrompt();
  }

  const inputPickedPath = await pickDirectoryPathFromInput();
  if (inputPickedPath) {
    return inputPickedPath;
  }
  return pickDirectoryPathFromPrompt();
}

async function tryPickDirectoryPathFromTauri(): Promise<DirectoryPickAttempt> {
  try {
    const picked = await openDialog({
      directory: true,
      multiple: false,
      title: "选择项目目录"
    });
    return {
      handled: true,
      value: normalizePickedPath(picked)
    };
  } catch {
    return { handled: false, value: null };
  }
}

function normalizePickedPath(value: string | string[] | null): string | null {
  if (Array.isArray(value)) {
    return normalizeSinglePath(value[0] ?? "");
  }
  return normalizeSinglePath(value ?? "");
}

function normalizeSinglePath(input: string): string | null {
  const normalized = normalizePath(input);
  return normalized === "" ? null : normalized;
}

function pickDirectoryPathFromPrompt(): string | null {
  if (typeof window === "undefined" || typeof window.prompt !== "function") {
    return null;
  }

  const manualPath = window.prompt("未读取到目录。如需导入空目录，请输入目录绝对路径：", "");
  if (manualPath === null) {
    return null;
  }

  return normalizeSinglePath(manualPath);
}

function pickDirectoryPathFromInput(): Promise<string | null> {
  return new Promise((resolve) => {
    const input = document.createElement("input");
    input.type = "file";
    input.multiple = true;
    input.setAttribute("webkitdirectory", "");
    input.setAttribute("directory", "");
    input.style.position = "fixed";
    input.style.left = "-9999px";

    let settled = false;
    let changeObserved = false;
    let focusFallbackTimer: number | null = null;

    const finalize = (value: string | null): void => {
      if (settled) {
        return;
      }
      settled = true;
      if (focusFallbackTimer !== null) {
        window.clearTimeout(focusFallbackTimer);
        focusFallbackTimer = null;
      }
      input.removeEventListener("change", onChange);
      window.removeEventListener("focus", onWindowFocus, true);
      if (input.parentNode) {
        input.parentNode.removeChild(input);
      }
      resolve(value);
    };

    const onChange = (): void => {
      changeObserved = true;
      finalize(extractDirectoryPath(input.files));
    };

    const onWindowFocus = (): void => {
      if (focusFallbackTimer !== null) {
        window.clearTimeout(focusFallbackTimer);
      }
      focusFallbackTimer = window.setTimeout(() => {
        if (settled) {
          return;
        }
        if (changeObserved) {
          return;
        }
        finalize(extractDirectoryPath(input.files));
      }, FOCUS_FALLBACK_DELAY_MS);
    };

    input.addEventListener("change", onChange);
    window.addEventListener("focus", onWindowFocus, true);
    document.body.appendChild(input);
    input.click();
  });
}

function extractDirectoryPath(files: FileList | null): string | null {
  if (!files || files.length === 0) {
    return null;
  }

  const firstFile = files.item(0) as DesktopDirectoryFile | null;
  if (!firstFile) {
    return null;
  }

  const absoluteFilePath = normalizePath(firstFile.path ?? "");
  const relativeParts = splitPath(firstFile.webkitRelativePath ?? "");

  if (absoluteFilePath !== "") {
    const removeSegments = relativeParts.length >= 2 ? relativeParts.length - 1 : 1;
    const directoryPath = trimPathSegments(absoluteFilePath, removeSegments);
    if (directoryPath !== "") {
      return directoryPath;
    }
  }

  return relativeParts[0] ?? null;
}

function normalizePath(input: string): string {
  return input.trim().replace(/\\/g, "/");
}

function splitPath(input: string): string[] {
  return normalizePath(input)
    .split("/")
    .filter((part) => part !== "");
}

function trimPathSegments(path: string, count: number): string {
  if (count <= 0) {
    return path;
  }

  const parts = normalizePath(path).split("/");
  if (parts.length <= count) {
    return "";
  }
  const kept = parts.slice(0, parts.length - count);
  if (kept.length === 1 && kept[0] === "") {
    return "/";
  }
  return kept.join("/");
}
