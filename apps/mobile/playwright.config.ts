import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "@playwright/test";

const configDir = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  expect: {
    timeout: 5_000
  },
  fullyParallel: false,
  workers: 1,
  reporter: "list",
  use: {
    baseURL: "http://127.0.0.1:4173",
    viewport: { width: 390, height: 844 },
    trace: "on-first-retry"
  },
  webServer: {
    command: "VITE_HUB_BASE_URL=http://127.0.0.1:9 pnpm dev --host 127.0.0.1 --port 4173",
    cwd: configDir,
    port: 4173,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000
  }
});
