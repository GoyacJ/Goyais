#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..", "..");
const desktopSrcPath = path.join(repoRoot, "apps/desktop/src");
const tokensPath = path.join(repoRoot, "apps/desktop/src/styles/tokens.css");
const penPath = path.join(repoRoot, "apps/desktop/design/goyais-v0.4.0-design-system.pen");

const requiredTokens = [
  "--global-space-6",
  "--global-space-10",
  "--global-space-20",
  "--global-radius-4",
  "--global-radius-6",
  "--global-font-family-code",
  "--semantic-bg",
  "--semantic-surface",
  "--semantic-surface-2",
  "--semantic-overlay",
  "--semantic-link",
  "--semantic-text",
  "--semantic-text-muted",
  "--semantic-border",
  "--semantic-divider",
  "--component-button-primary-bg",
  "--component-input-bg",
  "--component-sidebar-bg",
  "--component-topbar-bg",
  "--component-modal-bg",
  "--component-toast-error-bg",
  "--component-table-bg",
  "--component-list-item-bg-hover"
];

const forbiddenPatterns = [/#0f172a/gi, /#1e3a8a/gi];
const allowedStyleExts = new Set([".css", ".vue"]);
const hardcodeSkipFiles = new Set([
  path.join(repoRoot, "apps/desktop/src/styles/base.css"),
  path.join(repoRoot, "apps/desktop/src/styles/theme-profiles.css"),
  tokensPath
]);

function main() {
  const errors = [];

  if (!fs.existsSync(tokensPath)) {
    errors.push(`tokens file not found: ${tokensPath}`);
  }
  if (!fs.existsSync(penPath)) {
    errors.push(`pen file not found: ${penPath}`);
  }
  if (!fs.existsSync(desktopSrcPath)) {
    errors.push(`desktop src path not found: ${desktopSrcPath}`);
  }
  if (errors.length > 0) {
    exitWithErrors(errors);
  }

  const tokensText = fs.readFileSync(tokensPath, "utf8");
  const tokenDefinitions = collectTokenDefinitions(tokensText);

  for (const token of requiredTokens) {
    if (!tokenDefinitions.has(token)) {
      errors.push(`missing token: ${token}`);
    }
  }

  if (!tokensText.includes("--semantic-bg: var(--global-color-neutral-900);")) {
    errors.push("semantic bg is not mapped to neutral-900 (#181818ff)");
  }
  if (!tokensText.includes("--semantic-surface: var(--global-color-neutral-800);")) {
    errors.push("semantic surface is not mapped to neutral-800 (#1f1f1fff)");
  }

  for (const pattern of forbiddenPatterns) {
    if (pattern.test(tokensText)) {
      errors.push(`forbidden legacy color detected: ${pattern}`);
    }
  }

  const styleFiles = collectStyleFiles(desktopSrcPath);
  const undefinedRefs = checkUndefinedTokenReferences(styleFiles, tokenDefinitions);
  const hardcodeViolations = checkHardcodedStyleValues(styleFiles);
  errors.push(...undefinedRefs, ...hardcodeViolations);

  if (errors.length > 0) {
    exitWithErrors(errors);
  }

  console.log("[check-token-drift] OK");
  console.log(`- tokens: ${tokensPath}`);
  console.log(`- design: ${penPath}`);
}

function collectTokenDefinitions(cssText) {
  const definitions = new Set();
  const definitionPattern = /(--[a-z0-9-]+)\s*:/gi;
  let match = definitionPattern.exec(cssText);
  while (match) {
    definitions.add(match[1]);
    match = definitionPattern.exec(cssText);
  }
  return definitions;
}

function collectStyleFiles(rootPath) {
  const files = [];
  const stack = [rootPath];

  while (stack.length > 0) {
    const current = stack.pop();
    if (!current) {
      continue;
    }
    const stat = fs.statSync(current);
    if (stat.isDirectory()) {
      const entries = fs.readdirSync(current, { withFileTypes: true });
      for (const entry of entries) {
        stack.push(path.join(current, entry.name));
      }
      continue;
    }

    const ext = path.extname(current).toLowerCase();
    if (allowedStyleExts.has(ext)) {
      files.push(current);
    }
  }

  return files;
}

function checkUndefinedTokenReferences(files, tokenDefinitions) {
  const issues = [];

  for (const filePath of files) {
    const fileText = fs.readFileSync(filePath, "utf8");
    const localDefinitions = collectTokenDefinitions(fileText);
    const styleRanges = collectStyleRanges(filePath, fileText);

    for (const range of styleRanges) {
      const refPattern = /var\(\s*(--[a-z0-9-]+)\s*(?:,[^)]+)?\)/gi;
      let match = refPattern.exec(range.content);
      while (match) {
        const variableName = match[1];
        if (!tokenDefinitions.has(variableName) && !localDefinitions.has(variableName)) {
          const absoluteIndex = range.startIndex + (match.index ?? 0);
          const line = lineAtIndex(fileText, absoluteIndex);
          issues.push(
            `${path.relative(repoRoot, filePath)}:${line} references undefined token "${variableName}"`
          );
        }
        match = refPattern.exec(range.content);
      }
    }
  }

  return issues;
}

function checkHardcodedStyleValues(files) {
  const issues = [];
  const colorLiteralPattern = /#([0-9a-f]{3,8})\b|rgba?\(|hsla?\(|oklch?\(/i;
  const spacingRadiusNumericPattern = /-?\d*\.?\d+(px|rem|em|%)\b/i;
  const propPattern = /^\s*([a-z-]+)\s*:\s*([^;]+);/i;

  const colorProps = new Set([
    "color",
    "background",
    "background-color",
    "border-color",
    "outline-color",
    "fill",
    "stroke",
    "box-shadow"
  ]);
  const fontProps = new Set(["font-family", "font-size"]);
  const spacingRadiusProps = new Set([
    "margin",
    "margin-top",
    "margin-right",
    "margin-bottom",
    "margin-left",
    "padding",
    "padding-top",
    "padding-right",
    "padding-bottom",
    "padding-left",
    "gap",
    "row-gap",
    "column-gap",
    "border-radius",
    "border-top-left-radius",
    "border-top-right-radius",
    "border-bottom-left-radius",
    "border-bottom-right-radius"
  ]);

  for (const filePath of files) {
    if (hardcodeSkipFiles.has(filePath)) {
      continue;
    }

    const fileText = fs.readFileSync(filePath, "utf8");
    const styleRanges = collectStyleRanges(filePath, fileText);

    for (const range of styleRanges) {
      const lines = range.content.split(/\r?\n/);
      for (let i = 0; i < lines.length; i += 1) {
        const line = lines[i];
        if (line.trimStart().startsWith("/*") || line.trimStart().startsWith("*")) {
          continue;
        }
        const propMatch = propPattern.exec(line);
        if (!propMatch) {
          continue;
        }
        const prop = propMatch[1].toLowerCase();
        const value = propMatch[2].trim();
        if (value.includes("var(")) {
          continue;
        }

        const lineNumber = lineAtIndex(fileText, range.startIndex + indexBeforeLine(lines, i));
        const relativePath = path.relative(repoRoot, filePath);

        if (colorProps.has(prop) && colorLiteralPattern.test(value)) {
          issues.push(`${relativePath}:${lineNumber} has hardcoded color value "${value}"`);
          continue;
        }

        if (fontProps.has(prop)) {
          issues.push(`${relativePath}:${lineNumber} has hardcoded ${prop} value "${value}"`);
          continue;
        }

        if (spacingRadiusProps.has(prop) && spacingRadiusNumericPattern.test(value)) {
          issues.push(`${relativePath}:${lineNumber} has hardcoded ${prop} value "${value}"`);
        }
      }
    }
  }

  return issues;
}

function collectStyleRanges(filePath, fileText) {
  if (path.extname(filePath).toLowerCase() === ".css") {
    return [{ content: fileText, startIndex: 0 }];
  }

  const ranges = [];
  const stylePattern = /<style\b[^>]*>([\s\S]*?)<\/style>/gi;
  let match = stylePattern.exec(fileText);
  while (match) {
    const fullMatch = match[0];
    const content = match[1] ?? "";
    const innerOffset = fullMatch.indexOf(content);
    const startIndex = (match.index ?? 0) + Math.max(innerOffset, 0);
    ranges.push({ content, startIndex });
    match = stylePattern.exec(fileText);
  }
  return ranges;
}

function lineAtIndex(text, index) {
  if (index <= 0) {
    return 1;
  }
  const slice = text.slice(0, index);
  return slice.split(/\r?\n/).length;
}

function indexBeforeLine(lines, lineIndex) {
  let offset = 0;
  for (let i = 0; i < lineIndex; i += 1) {
    offset += lines[i].length + 1;
  }
  return offset;
}

function exitWithErrors(errors) {
  console.error("[check-token-drift] FAILED");
  for (const error of errors) {
    console.error(`- ${error}`);
  }
  process.exit(1);
}

main();
