import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { pickDirectoryPath } from "@/shared/services/directoryPicker";

type MutableDesktopFile = File & {
  path?: string;
  webkitRelativePath?: string;
};

describe("pickDirectoryPath", () => {
  beforeEach(() => {
    vi.useFakeTimers();
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

    const pending = pickDirectoryPath();
    await vi.advanceTimersByTimeAsync(500);

    await expect(pending).resolves.toBeNull();
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
