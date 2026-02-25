import path from "node:path";

import vue from "@vitejs/plugin-vue";
import UnoCSS from "unocss/vite";
import { defineConfig } from "vite";

const sharedSourceRoot = path.resolve(__dirname, "../desktop/src");

export default defineConfig({
  plugins: [vue(), UnoCSS()],
  resolve: {
    alias: {
      "@": sharedSourceRoot
    }
  },
  server: {
    fs: {
      allow: [path.resolve(__dirname, "..")]
    }
  }
});
