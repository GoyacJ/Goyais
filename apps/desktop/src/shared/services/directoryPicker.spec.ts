import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const dialogMocks = vi.hoisted(() => ({
  open: vi.fn()
}));

vi.mock("@tauri-apps/plugin-dialog", () => ({
  open: dialogMocks.open
}));

import { pickDirectoryPath } from "@/shared/services/directoryPicker";

type MutableDesktopFile = File & {
  path?: string;
  webkitRelativePath?: string;
};

describe("pickDirectoryPath", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    dialogMocks.open.mockReset();
    dialogMocks.open.mockRejectedValue(new Error("tauri dialog unavailable"));
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
    document.body.innerHTML = "";
  });

  it("resolves selected directory when focus fires before change", async () => {
    vi.spyOn(HTMLInputElement.prototype, "click").mockImplementation(function (this: HTMLInputElement) {
      const input = this;

      window.dispatchEvent(new Event("focus"));
      window.setTimeout(() => {
        const file = createDesktopFile("/tmp/repo-alpha/main.ts", "repo-alpha/main.ts");
        setInputFiles(input, [file]);
        input.dispatchEvent(new Event("change"));
      }, 10);
    });

    const pending = pickDirectoryPath();
    await vi.advanceTimersByTimeAsync(20);

    await expect(pending).resolves.toBe("/tmp/repo-alpha");
  });

  it("returns null when picker closes without choosing directory", async () => {
    vi.spyOn(HTMLInputElement.prototype, "click").mockImplementation(() => {
      window.dispatchEvent(new Event("focus"));
    });
    vi.spyOn(window, "prompt").mockReturnValueOnce(null);

    const pending = pickDirectoryPath();
    await vi.advanceTimersByTimeAsync(1500);

    await expect(pending).resolves.toBeNull();
  });

  it("falls back to manual directory path input when picker cannot resolve files", async () => {
    vi.spyOn(HTMLInputElement.prototype, "click").mockImplementation(() => {
      window.dispatchEvent(new Event("focus"));
    });
    vi.spyOn(window, "prompt").mockReturnValueOnce("/tmp/empty-project");

    const pending = pickDirectoryPath();
    await vi.advanceTimersByTimeAsync(1500);

    await expect(pending).resolves.toBe("/tmp/empty-project");
  });

  it("uses tauri dialog result when available (supports empty directory selection)", async () => {
    dialogMocks.open.mockResolvedValue("/tmp/empty-folder");
    const promptSpy = vi.spyOn(window, "prompt").mockReturnValueOnce("/tmp/should-not-be-used");

    await expect(pickDirectoryPath()).resolves.toBe("/tmp/empty-folder");
    expect(promptSpy).not.toHaveBeenCalled();
  });
});

function createDesktopFile(path: string, relativePath: string): MutableDesktopFile {
  const file = new File(["content"], "main.ts") as MutableDesktopFile;
  Object.defineProperty(file, "path", { configurable: true, value: path });
  Object.defineProperty(file, "webkitRelativePath", { configurable: true, value: relativePath });
  return file;
}

function setInputFiles(input: HTMLInputElement, files: File[]): void {
  const filesLike = {
    item: (index: number) => files[index] ?? null
  } as FileList & Record<number, File>;
  files.forEach((file, index) => {
    filesLike[index] = file;
  });
  Object.defineProperty(filesLike, "length", {
    configurable: true,
    value: files.length
  });
  Object.defineProperty(input, "files", {
    configurable: true,
    value: filesLike
  });
}
