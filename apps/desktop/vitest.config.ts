import path from "node:path";

import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src")
    }
  },
  test: {
    environment: "jsdom",
    globals: true,
    exclude: ["e2e/**"],
    coverage: {
      provider: "v8",
      reportsDirectory: "./coverage",
      reporter: ["text", "json-summary"],
      include: ["src/**/*.{ts,vue}"],
      exclude: [
        "src/**/*.spec.ts",
        "src/**/*.d.ts",
        "src/App.vue",
        "src/main.ts",
        "src/shared/types/**",
        "src/shared/services/windowControls.ts",
        "src/shared/services/sseClient.ts",
        "src/modules/admin/**",
        "src/modules/project/schemas/**",
        "src/modules/project/services/**",
        "src/modules/project/store/projectActions.ts",
        "src/modules/resource/schemas/**",
        "src/modules/workspace/schemas/settingsContent.ts",
        "src/modules/workspace/services/index.ts",
        "src/modules/conversation/views/useMainScreen*.ts",
        "src/modules/conversation/views/streamCoordinator.ts"
      ]
    }
  }
});
