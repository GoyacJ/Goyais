type DesktopDirectoryFile = File & {
  path?: string;
  webkitRelativePath?: string;
};

const FOCUS_FALLBACK_DELAY_MS = 260;

export function pickDirectoryPath(): Promise<string | null> {
  if (typeof document === "undefined" || typeof window === "undefined") {
    return Promise.resolve(null);
  }

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
