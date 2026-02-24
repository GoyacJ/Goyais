#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..", "..");
const desktopSrcPath = path.join(repoRoot, "apps/desktop/src");

const thresholdsByExt = new Map([
  [".go", 400],
  [".py", 350],
  [".ts", 300],
  [".tsx", 300],
  [".vue", 300],
  [".rs", 350]
]);

const ignoredFilePatterns = [/\.spec\.[cm]?[jt]sx?$/i, /\/tests\//, /\.d\.ts$/i];

function main() {
  if (!fs.existsSync(desktopSrcPath)) {
    fail([`desktop src path not found: ${desktopSrcPath}`]);
  }

  const violations = [];
  const files = walkFiles(desktopSrcPath);
  for (const filePath of files) {
    const ext = path.extname(filePath).toLowerCase();
    const threshold = thresholdsByExt.get(ext);
    if (!threshold) {
      continue;
    }
    const normalized = filePath.replaceAll(path.sep, "/");
    if (ignoredFilePatterns.some((pattern) => pattern.test(normalized))) {
      continue;
    }

    const content = fs.readFileSync(filePath, "utf8");
    const lineCount = content.split(/\r?\n/).length;
    if (lineCount > threshold) {
      violations.push({
        filePath,
        lineCount,
        threshold
      });
    }
  }

  if (violations.length > 0) {
    const lines = violations
      .sort((a, b) => b.lineCount - a.lineCount)
      .map((item) => {
        const relativePath = path.relative(repoRoot, item.filePath);
        return `${relativePath} => ${item.lineCount} lines (threshold ${item.threshold})`;
      });
    fail(lines);
  }

  console.log("[check-file-size] OK");
  console.log(`- scanned: ${files.length} files`);
}

function walkFiles(startPath) {
  const files = [];
  const stack = [startPath];
  while (stack.length > 0) {
    const currentPath = stack.pop();
    if (!currentPath) {
      continue;
    }
    const stat = fs.statSync(currentPath);
    if (stat.isDirectory()) {
      const entries = fs.readdirSync(currentPath, { withFileTypes: true });
      for (const entry of entries) {
        stack.push(path.join(currentPath, entry.name));
      }
      continue;
    }
    files.push(currentPath);
  }
  return files;
}

function fail(lines) {
  console.error("[check-file-size] FAILED");
  for (const line of lines) {
    console.error(`- ${line}`);
  }
  process.exit(1);
}

main();
