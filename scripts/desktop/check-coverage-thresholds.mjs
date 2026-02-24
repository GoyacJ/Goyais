#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..", "..");
const coverageSummaryPath = path.join(repoRoot, "apps/desktop/coverage/coverage-summary.json");

const overallThreshold = 70;
const coreThreshold = 80;

const coreModuleMatchers = [
  /apps\/desktop\/src\/shared\/stores\/permissionStore\.ts$/,
  /apps\/desktop\/src\/shared\/services\/permissionService\.ts$/,
  /apps\/desktop\/src\/modules\/resource\//,
  /apps\/desktop\/src\/modules\/conversation\/store\/executionActions\.ts$/,
  /apps\/desktop\/src\/modules\/conversation\/store\/executionRuntime\.ts$/,
  /apps\/desktop\/src\/modules\/conversation\/store\/state\.ts$/
];

function main() {
  if (!fs.existsSync(coverageSummaryPath)) {
    fail([
      `coverage summary not found: ${coverageSummaryPath}`,
      "run `pnpm --filter @goyais/desktop coverage` before check:coverage"
    ]);
  }

  const summary = JSON.parse(fs.readFileSync(coverageSummaryPath, "utf8"));
  const errors = [];

  const total = summary.total;
  if (!total) {
    fail(["coverage summary missing 'total' section"]);
  }

  checkMetric("overall lines", total.lines?.pct ?? 0, overallThreshold, errors);
  checkMetric("overall statements", total.statements?.pct ?? 0, overallThreshold, errors);
  checkMetric("overall functions", total.functions?.pct ?? 0, overallThreshold, errors);
  checkMetric("overall branches", total.branches?.pct ?? 0, overallThreshold, errors);

  const coreFiles = Object.keys(summary).filter((key) => key !== "total" && coreModuleMatchers.some((matcher) => matcher.test(key.replaceAll("\\", "/"))));
  if (coreFiles.length === 0) {
    errors.push("no core module coverage entries found in coverage summary");
  } else {
    const coreAggregate = aggregateMetrics(summary, coreFiles);
    checkMetric("core lines", coreAggregate.lines, coreThreshold, errors);
    checkMetric("core statements", coreAggregate.statements, coreThreshold, errors);
    checkMetric("core functions", coreAggregate.functions, coreThreshold, errors);
    checkMetric("core branches", coreAggregate.branches, coreThreshold, errors);
  }

  if (errors.length > 0) {
    fail(errors);
  }

  console.log("[check-coverage-thresholds] OK");
  console.log(`- summary: ${coverageSummaryPath}`);
}

function aggregateMetrics(summary, filePaths) {
  const counters = {
    lines: { covered: 0, total: 0 },
    statements: { covered: 0, total: 0 },
    functions: { covered: 0, total: 0 },
    branches: { covered: 0, total: 0 }
  };

  for (const filePath of filePaths) {
    const entry = summary[filePath];
    for (const key of Object.keys(counters)) {
      counters[key].covered += entry[key]?.covered ?? 0;
      counters[key].total += entry[key]?.total ?? 0;
    }
  }

  return {
    lines: percentage(counters.lines.covered, counters.lines.total),
    statements: percentage(counters.statements.covered, counters.statements.total),
    functions: percentage(counters.functions.covered, counters.functions.total),
    branches: percentage(counters.branches.covered, counters.branches.total)
  };
}

function percentage(covered, total) {
  if (total <= 0) {
    return 0;
  }
  return Number(((covered / total) * 100).toFixed(2));
}

function checkMetric(name, actual, threshold, errors) {
  if (actual < threshold) {
    errors.push(`${name} ${actual}% < threshold ${threshold}%`);
  }
}

function fail(lines) {
  console.error("[check-coverage-thresholds] FAILED");
  for (const line of lines) {
    console.error(`- ${line}`);
  }
  process.exit(1);
}

main();
