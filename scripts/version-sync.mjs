#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(dirname, "..");

const args = process.argv.slice(2).filter((arg) => arg !== "--");
const semverPattern = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/;

let mode = "sync";
let setVersion;

if (args[0] === "--check") {
  mode = "check";
} else if (args[0] === "--set") {
  mode = "sync";
  setVersion = args[1];
  if (!setVersion) {
    console.error("Missing version value. Usage: node scripts/version-sync.mjs --set <version>");
    process.exit(1);
  }
  if (!semverPattern.test(setVersion)) {
    console.error(`Invalid semantic version: ${setVersion}`);
    process.exit(1);
  }
} else if (args.length > 0) {
  console.error(`Unknown arguments: ${args.join(" ")}`);
  console.error("Usage: node scripts/version-sync.mjs [--check] [--set <version>]");
  process.exit(1);
}

function absolutePath(relPath) {
  return path.join(repoRoot, relPath);
}

function readText(relPath) {
  return fs.readFileSync(absolutePath(relPath), "utf8");
}

function writeText(relPath, value) {
  fs.writeFileSync(absolutePath(relPath), value);
}

function readJson(relPath) {
  return JSON.parse(readText(relPath));
}

function writeJson(relPath, value) {
  writeText(relPath, `${JSON.stringify(value, null, 2)}\n`);
}

function replaceChecked(file, source, pattern, replacement, label) {
  const matched = source.match(pattern);
  if (!matched) {
    throw new Error(`${file}: could not find ${label}`);
  }
  return source.replace(pattern, replacement);
}

function replaceAllChecked(file, source, pattern, replacement, label, minCount = 1) {
  const matches = source.match(pattern);
  if (!matches || matches.length < minCount) {
    throw new Error(`${file}: expected at least ${minCount} match(es) for ${label}, got ${matches?.length ?? 0}`);
  }
  return source.replace(pattern, replacement);
}

const changed = [];
const mismatches = [];

function trackChange(relPath, before, after) {
  if (before === after) {
    return;
  }
  if (mode === "sync") {
    writeText(relPath, after);
    changed.push(relPath);
    return;
  }
  mismatches.push(relPath);
}

const jsonVersionFiles = [
  "package.json",
  "apps/desktop-tauri/package.json",
  "apps/desktop-tauri/src-tauri/tauri.conf.json",
  "server/hub-server/package.json",
  "server/sync-server/package.json",
  "runtime/python-agent/package.json",
  "packages/protocol/package.json"
];

if (setVersion) {
  const rootPackage = readJson("package.json");
  rootPackage.version = setVersion;
  writeJson("package.json", rootPackage);
}

const sourceVersion = readJson("package.json").version;
if (!semverPattern.test(sourceVersion)) {
  console.error(`Root package.json has invalid version: ${sourceVersion}`);
  process.exit(1);
}

for (const relPath of jsonVersionFiles) {
  const json = readJson(relPath);
  if (typeof json.version !== "string") {
    throw new Error(`${relPath}: missing "version" field`);
  }
  const next = { ...json, version: sourceVersion };
  const before = `${JSON.stringify(json, null, 2)}\n`;
  const after = `${JSON.stringify(next, null, 2)}\n`;
  trackChange(relPath, before, after);
}

{
  const relPath = "apps/desktop-tauri/src-tauri/Cargo.toml";
  const before = readText(relPath);
  const after = replaceChecked(
    relPath,
    before,
    /(\[package\][\s\S]*?\nversion = ")[^"]+(")/,
    `$1${sourceVersion}$2`,
    "[package].version"
  );
  trackChange(relPath, before, after);
}

{
  const relPath = "runtime/python-agent/pyproject.toml";
  const before = readText(relPath);
  const after = replaceChecked(
    relPath,
    before,
    /(\[project\][\s\S]*?\nversion = ")[^"]+(")/,
    `$1${sourceVersion}$2`,
    "[project].version"
  );
  trackChange(relPath, before, after);
}

{
  const relPath = "packages/protocol/generated/python/pyproject.toml";
  const before = readText(relPath);
  const after = replaceChecked(
    relPath,
    before,
    /(\[project\][\s\S]*?\nversion = ")[^"]+(")/,
    `$1${sourceVersion}$2`,
    "[project].version"
  );
  trackChange(relPath, before, after);
}

{
  const relPath = "server/hub-server/src/app.ts";
  const before = readText(relPath);
  const after = replaceAllChecked(
    relPath,
    before,
    /version:\s*"[^"]+"/g,
    `version: "${sourceVersion}"`,
    "version fields",
    2
  );
  trackChange(relPath, before, after);
}

{
  const relPath = "server/sync-server/src/app.ts";
  const before = readText(relPath);
  const after = replaceChecked(
    relPath,
    before,
    /runtime_version:\s*"[^"]+"/,
    `runtime_version: "${sourceVersion}"`,
    "runtime_version field"
  );
  trackChange(relPath, before, after);
}

{
  const relPath = "runtime/python-agent/app/main.py";
  const before = readText(relPath);
  const after = replaceChecked(
    relPath,
    before,
    /version="[^"]+"/,
    `version="${sourceVersion}"`,
    "FastAPI version"
  );
  trackChange(relPath, before, after);
}

{
  const relPath = "runtime/python-agent/app/api/ops.py";
  const before = readText(relPath);
  const withHealth = replaceChecked(
    relPath,
    before,
    /"version":\s*"[^"]+"/,
    `"version": "${sourceVersion}"`,
    "health version field"
  );
  const after = replaceChecked(
    relPath,
    withHealth,
    /"runtime_version":\s*"[^"]+"/,
    `"runtime_version": "${sourceVersion}"`,
    "runtime version field"
  );
  trackChange(relPath, before, after);
}

{
  const relPath = "server/hub-server-go/internal/router/router.go";
  const before = readText(relPath);
  const after = replaceChecked(
    relPath,
    before,
    /NewHealthHandler\("[^"]+"\)/,
    `NewHealthHandler("${sourceVersion}")`,
    "NewHealthHandler version argument"
  );
  trackChange(relPath, before, after);
}

{
  const relPath = "apps/desktop-tauri/src-tauri/Cargo.lock";
  const before = readText(relPath);
  const after = replaceChecked(
    relPath,
    before,
    /(\[\[package\]\]\nname = "goyais-desktop"\nversion = ")[^"]+(")/,
    `$1${sourceVersion}$2`,
    "goyais-desktop package version"
  );
  trackChange(relPath, before, after);
}

{
  const relPath = "runtime/python-agent/uv.lock";
  const before = readText(relPath);
  const after = replaceChecked(
    relPath,
    before,
    /(\[\[package\]\]\nname = "goyais-python-agent"\nversion = ")[^"]+(")/,
    `$1${sourceVersion}$2`,
    "goyais-python-agent package version"
  );
  trackChange(relPath, before, after);
}

if (mode === "check") {
  if (mismatches.length > 0) {
    console.error(`Version mismatch detected (source: ${sourceVersion}):`);
    for (const file of mismatches) {
      console.error(`- ${file}`);
    }
    process.exit(1);
  }
  console.log(`All versioned files are aligned to ${sourceVersion}.`);
  process.exit(0);
}

if (changed.length === 0) {
  console.log(`No changes needed. All versioned files already at ${sourceVersion}.`);
} else {
  console.log(`Synchronized ${changed.length} file(s) to version ${sourceVersion}:`);
  for (const file of changed) {
    console.log(`- ${file}`);
  }
}
