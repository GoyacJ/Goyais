#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..", "..");
const tokensPath = path.join(repoRoot, "apps/desktop/src/styles/tokens.css");
const penPath = path.join(repoRoot, "apps/desktop/design/goyais-v0.4.0-design-system.pen");

const requiredTokens = [
  "--semantic-bg",
  "--semantic-surface",
  "--semantic-surface-2",
  "--semantic-overlay",
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

function main() {
  const errors = [];

  if (!fs.existsSync(tokensPath)) {
    errors.push(`tokens file not found: ${tokensPath}`);
  }
  if (!fs.existsSync(penPath)) {
    errors.push(`pen file not found: ${penPath}`);
  }
  if (errors.length > 0) {
    exitWithErrors(errors);
  }

  const tokensText = fs.readFileSync(tokensPath, "utf8");

  for (const token of requiredTokens) {
    if (!tokensText.includes(token)) {
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

  if (errors.length > 0) {
    exitWithErrors(errors);
  }

  console.log("[check-token-drift] OK");
  console.log(`- tokens: ${tokensPath}`);
  console.log(`- design: ${penPath}`);
}

function exitWithErrors(errors) {
  console.error("[check-token-drift] FAILED");
  for (const error of errors) {
    console.error(`- ${error}`);
  }
  process.exit(1);
}

main();
